package ssh

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// SSHManager SSH连接管理器
type SSHManager struct {
	config *ssh.ClientConfig
}

// NewSSHManager 创建SSH管理器
func NewSSHManager(username, password string) *SSHManager {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
	}
	return &SSHManager{
		config: config,
	}
}

// Connect 连接到SSH服务器
func (m *SSHManager) Connect(host string, port int) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, m.config)
}

// Session 创建SSH会话
func (m *SSHManager) Session(client *ssh.Client) (*ssh.Session, error) {
	return client.NewSession()
}

// Close 关闭SSH连接
func (m *SSHManager) Close(client *ssh.Client) {
	if client != nil {
		client.Close()
	}
}

// SSHConnection SSH连接包装器
type SSHConnection struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
}

// NewSSHConnection 创建SSH连接
func NewSSHConnection(client *ssh.Client) (*SSHConnection, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED:  14400,
		ssh.TTY_OP_OSPEED:  14400,
	}

	// 请求伪终端
	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		session.Close()
		return nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	return &SSHConnection{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// GetStdin 获取标准输入
func (c *SSHConnection) GetStdin() io.WriteCloser {
	return c.stdin
}

// GetStdout 获取标准输出
func (c *SSHConnection) GetStdout() io.Reader {
	return c.stdout
}

// GetStderr 获取标准错误输出
func (c *SSHConnection) GetStderr() io.Reader {
	return c.stderr
}

// Close 关闭SSH连接
func (c *SSHConnection) Close() {
	if c.session != nil {
		c.session.Close()
	}
	if c.client != nil {
		c.client.Close()
	}
}