package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BackupStorage struct {
	basePath string
}

func NewBackupStorage(basePath string) *BackupStorage {
	return &BackupStorage{basePath: basePath}
}

// CreateBackupDirectory 创建备份目录
func (s *BackupStorage) CreateBackupDirectory(clusterID, backupName string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(s.basePath, clusterID, backupName, timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	return backupPath, nil
}

// WriteData 写入备份数据
func (s *BackupStorage) WriteData(backupPath, filename string, data []byte) error {
	filePath := filepath.Join(backupPath, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// CompressDirectory 压缩目录
func (s *BackupStorage) CompressDirectory(sourceDir, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(sourceDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath := strings.TrimPrefix(file, sourceDir)
		if relPath == "." || relPath == "" {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, relPath)
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if !fi.IsDir() {
			file, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to copy file data: %w", err)
			}
		}

		return nil
	})
}

// DecompressDirectory 解压缩目录
func (s *BackupStorage) DecompressDirectory(compressedFile, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 打开压缩文件
	file, err := os.Open(compressedFile)
	if err != nil {
		return fmt.Errorf("failed to open compressed file: %w", err)
	}
	defer file.Close()

	// 创建gzip读取器
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// 创建tar读取器
	tr := tar.NewReader(gzr)

	// 逐个解压文件
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// 构建目标路径
		targetPath := filepath.Join(outputDir, header.Name)

		// 处理目录
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// 处理文件
		if header.Typeflag == tar.TypeReg {
			// 确保目录存在
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// 创建文件
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			// 复制数据
			if _, err := io.Copy(outFile, tr); err != nil {
				return fmt.Errorf("failed to copy file data: %w", err)
			}
		}
	}

	return nil
}

// CalculateDirectorySize 计算目录大小
func (s *BackupStorage) CalculateDirectorySize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// RemoveDirectory 删除目录
func (s *BackupStorage) RemoveDirectory(dirPath string) error {
	return os.RemoveAll(dirPath)
}

// PathExists 检查路径是否存在
func (s *BackupStorage) PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
