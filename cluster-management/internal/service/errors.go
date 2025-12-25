package service

import "errors"

var (
	ErrTenantAlreadyExists       = func(name string) error { return errors.New("租户已存在: " + name) }
	ErrTenantNotFound            = errors.New("租户不存在")
	ErrSystemTenantCannotBeDeleted = errors.New("系统租户不能删除")
	ErrEnvironmentAlreadyExists  = func(ns string) error { return errors.New("环境已存在: " + ns) }
	ErrEnvironmentNotFound      = errors.New("环境不存在")
	ErrApplicationAlreadyExists = func(name string) error { return errors.New("应用已存在: " + name) }
	ErrApplicationNotFound      = errors.New("应用不存在")
	ErrQuotaNotFound            = errors.New("配额不存在")
	ErrInvalidKubeConfig        = errors.New("无效的kubeconfig")
)
