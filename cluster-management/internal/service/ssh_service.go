package service

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SSHService SSH服务
type SSHService struct {
}

// NewSSHService 创建SSH服务
func NewSSHService() *SSHService {
	return &SSHService{}
}

// SSHClient SSH客户端配置
type SSHClient struct {
	client *ssh.Client
}

// Connect 连接到远程主机
func (s *SSHService) Connect(host, username, password string) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	fmt.Printf("[SSH] Connecting to %s@%s:22\n", username, host)

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial failed for %s: %w", host, err)
	}

	fmt.Printf("[SSH] Successfully connected to %s\n", host)

	return &SSHClient{client: client}, nil
}

// Close 关闭连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ExecuteCommand 执行远程命令
func (c *SSHClient) ExecuteCommand(command string) (string, error) {
	fmt.Printf("[SSH] Executing command: %s\n", command)

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		errMsg := fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
		fmt.Printf("[SSH] Command failed: %v\n", errMsg)
		return "", errMsg
	}

	fmt.Printf("[SSH] Command completed successfully\n")
	return stdout.String(), nil
}

// DownloadFile 下载远程文件到本地
func (c *SSHClient) DownloadFile(remoteFile, localFile string) error {
	fmt.Printf("[SSH] Downloading file from %s to %s\n", remoteFile, localFile)

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// 确保本地目录存在
	if err := os.MkdirAll(filepath.Dir(localFile), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// 检查远程文件是否存在
	if _, err := sftpClient.Stat(remoteFile); err != nil {
		return fmt.Errorf("remote file does not exist: %w", err)
	}

	// 打开远程文件
	remoteFileHandle, err := sftpClient.Open(remoteFile)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFileHandle.Close()

	// 创建本地文件
	localFileHandle, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFileHandle.Close()

	// 复制文件内容
	if _, err := io.Copy(localFileHandle, remoteFileHandle); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	fmt.Printf("[SSH] File downloaded successfully\n")
	return nil
}

// UploadFile 上传本地文件到远程
func (c *SSHClient) UploadFile(localFile, remoteFile string) error {
	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 打开本地文件
	localFileHandle, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer localFileHandle.Close()

	// 创建远程文件
	remoteFileHandle, err := sftpClient.Create(remoteFile)
	if err != nil {
		return err
	}
	defer remoteFileHandle.Close()

	// 复制文件内容
	if _, err := io.Copy(remoteFileHandle, localFileHandle); err != nil {
		return err
	}

	return nil
}
