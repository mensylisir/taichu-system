package repository

import (
	"github.com/taichu-system/cluster-management/internal/model"
	"gorm.io/gorm"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) Create(tenant *model.Tenant) error {
	return r.db.Create(tenant).Error
}

func (r *TenantRepository) GetByID(id string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.Where("id = ?", id).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) GetByName(name string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.Where("name = ?", name).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) List() ([]*model.Tenant, error) {
	var tenants []*model.Tenant
	err := r.db.Find(&tenants).Error
	return tenants, err
}

func (r *TenantRepository) ListAll() ([]*model.Tenant, error) {
	var tenants []*model.Tenant
	err := r.db.Find(&tenants).Error
	return tenants, err
}

func (r *TenantRepository) ListByType(tenantType string) ([]*model.Tenant, error) {
	var tenants []*model.Tenant
	err := r.db.Where("type = ?", tenantType).Find(&tenants).Error
	return tenants, err
}

func (r *TenantRepository) Update(tenant *model.Tenant) error {
	return r.db.Save(tenant).Error
}

func (r *TenantRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		tenant, err := r.GetByID(id)
		if err != nil {
			return err
		}

		if tenant.IsSystem {
			return gorm.ErrInvalidData
		}

		if err := tx.Exec("DELETE FROM applications WHERE tenant_id = ?", id).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM application_resource_specs WHERE application_id IN (SELECT id FROM applications WHERE tenant_id = ?)", id).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM resource_quotas WHERE environment_id IN (SELECT id FROM environments WHERE tenant_id = ?)", id).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM environments WHERE tenant_id = ?", id).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM tenant_quotas WHERE tenant_id = ?", id).Error; err != nil {
			return err
		}

		if err := tx.Delete(&model.Tenant{}, "id = ?", id).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *TenantRepository) EnsurePredefinedTenants() error {
	systemTenant, err := r.GetByName(model.PredefinedTenants.System)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			systemTenant = &model.Tenant{
				Name:        model.PredefinedTenants.System,
				DisplayName: "系统租户",
				Type:        model.TenantTypeSystem,
				Description: "系统预定义租户，用于承载系统组件",
				IsSystem:    true,
				Status:      model.TenantStatusActive,
			}
			if err := r.Create(systemTenant); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	defaultTenant, err := r.GetByName(model.PredefinedTenants.Default)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			defaultTenant = &model.Tenant{
				Name:        model.PredefinedTenants.Default,
				DisplayName: "默认租户",
				Type:        model.TenantTypeDefault,
				Description: "系统预定义租户，用于承载现有业务组件",
				IsSystem:    true,
				Status:      model.TenantStatusActive,
			}
			if err := r.Create(defaultTenant); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (r *TenantRepository) GetDB() *gorm.DB {
	return r.db
}
