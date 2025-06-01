package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

// Migrate runs database migrations
func Migrate() error {
	if DB == nil {
		return fmt.Errorf("database not connected")
	}

	log.Println("Running database migrations...")

	// Auto-migrate all models
	if err := DB.AutoMigrate(
		&models.Task{},
	); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Run custom migrations
	if err := runCustomMigrations(DB); err != nil {
		return fmt.Errorf("failed to run custom migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// runCustomMigrations runs any custom SQL migrations that can't be handled by AutoMigrate
func runCustomMigrations(db *gorm.DB) error {
	// Create indexes for better query performance
	migrations := []string{
		// Index on status for filtering tasks
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		
		// Index on repo for filtering tasks by repository
		`CREATE INDEX IF NOT EXISTS idx_tasks_repo ON tasks(repo)`,
		
		// Index on branch for finding tasks by branch
		`CREATE INDEX IF NOT EXISTS idx_tasks_branch ON tasks(branch)`,
		
		// Index on thread_id for Amp thread operations
		`CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON tasks(thread_id)`,
		
		// Index on created_at for chronological ordering
		`CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)`,
		
		// Index on updated_at for finding recently updated tasks
		`CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at)`,
		
		// Composite index for active tasks (non-terminal statuses)
		`CREATE INDEX IF NOT EXISTS idx_tasks_active ON tasks(status, updated_at) 
		 WHERE status IN ('queued', 'running', 'retrying', 'needs_review')`,
	}

	for _, migration := range migrations {
		if err := db.Exec(migration).Error; err != nil {
			return fmt.Errorf("failed to execute migration: %s, error: %w", migration, err)
		}
	}

	return nil
}

// DropAllTables drops all tables (useful for testing)
func DropAllTables() error {
	if DB == nil {
		return fmt.Errorf("database not connected")
	}

	// Drop tables in reverse dependency order
	tables := []interface{}{
		&models.Task{},
	}

	for _, table := range tables {
		if err := DB.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
	}

	return nil
}

// ResetDatabase drops all tables and recreates them (useful for testing)
func ResetDatabase() error {
	if err := DropAllTables(); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	if err := Migrate(); err != nil {
		return fmt.Errorf("failed to migrate after reset: %w", err)
	}

	return nil
}
