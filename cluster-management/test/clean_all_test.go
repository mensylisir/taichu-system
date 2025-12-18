package test

import (
	"fmt"
	"log"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCleanAll(t *testing.T) {
	dsn := "host=172.30.1.12 user=postgres password=Def@u1tpwd dbname=taichu port=32172 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 删除所有表（包括schema_migrations）
	tables := []string{
		"cluster_states", "nodes", "cluster_resources", "events",
		"security_policies", "autoscaling_policies", "audit_events",
		"cluster_expansions", "machines", "create_tasks", "clusters", "schema_migrations",
		"cluster_backups", "cluster_environments", "import_records", "backup_schedules",
	}

	// 先删除所有触发器
	db.Exec("DROP TRIGGER IF EXISTS update_nodes_updated_at ON nodes CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_cluster_resources_updated_at ON cluster_resources CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_events_updated_at ON events CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_security_policies_updated_at ON security_policies CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_autoscaling_policies_updated_at ON autoscaling_policies CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_clusters_updated_at ON clusters CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_cluster_states_updated_at ON cluster_states CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_cluster_backups_updated_at ON cluster_backups CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_cluster_environments_updated_at ON cluster_environments CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_import_records_updated_at ON import_records CASCADE")
	db.Exec("DROP TRIGGER IF EXISTS update_backup_schedules_updated_at ON backup_schedules CASCADE")

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			log.Printf("Failed to drop table %s: %v", table, err)
		} else {
			fmt.Printf("Dropped table: %s\n", table)
		}
	}

	// 删除所有索引
	db.Exec("DROP INDEX IF EXISTS idx_nodes_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_cluster_resources_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_events_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_security_policies_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_autoscaling_policies_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_cluster_backups_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_cluster_environments_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_import_records_cluster_id CASCADE")
	db.Exec("DROP INDEX IF EXISTS idx_backup_schedules_cluster_id CASCADE")
}
