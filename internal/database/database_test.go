package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brettsmith212/ci-test-2/internal/models"
)

// setupTestDB creates a test database for testing
func setupTestDB(t *testing.T) string {
	// Create test directory
	testDir := filepath.Join("../../testdata")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create unique test database file
	dbPath := filepath.Join(testDir, "test_"+t.Name()+".db")
	
	// Remove existing test database
	os.Remove(dbPath)
	
	return dbPath
}

// teardownTestDB cleans up test database
func teardownTestDB(t *testing.T, dbPath string) {
	if err := Close(); err != nil {
		t.Logf("Warning: Failed to close database: %v", err)
	}
	
	if err := os.Remove(dbPath); err != nil {
		t.Logf("Warning: Failed to remove test database: %v", err)
	}
}

func TestConnect(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Test successful connection
	err := Connect(dbPath)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Test that database instance is set
	if DB == nil {
		t.Fatal("Connect() did not set DB instance")
	}

	// Test that database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Connect() did not create database file")
	}
}

func TestConnect_InvalidPath(t *testing.T) {
	// Test connection with invalid path (read-only directory)
	invalidPath := "/root/invalid/path/test.db"
	err := Connect(invalidPath)
	if err == nil {
		t.Fatal("Connect() should fail with invalid path")
	}
}

func TestHealth(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Test health check before connection
	err := Health()
	if err == nil {
		t.Fatal("Health() should fail when database is not connected")
	}

	// Connect to database
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Test health check after connection
	err = Health()
	if err != nil {
		t.Fatalf("Health() failed: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Connect to database
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Run migrations
	err := Migrate()
	if err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}

	// Test that tables were created
	var count int64
	err = DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&count).Error
	if err != nil {
		t.Fatalf("Failed to check for tasks table: %v", err)
	}
	if count != 1 {
		t.Fatal("Migrate() did not create tasks table")
	}

	// Test that indexes were created
	err = DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name LIKE 'idx_tasks_%'").Scan(&count).Error
	if err != nil {
		t.Fatalf("Failed to check for indexes: %v", err)
	}
	if count == 0 {
		t.Fatal("Migrate() did not create indexes")
	}
}

func TestMigrate_WithoutConnection(t *testing.T) {
	// Reset DB to nil
	DB = nil

	err := Migrate()
	if err == nil {
		t.Fatal("Migrate() should fail when database is not connected")
	}
}

func TestResetDatabase(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Connect and migrate
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	if err := Migrate(); err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}

	// Create a test record
	task := &models.Task{
		ID:     "test-id",
		Repo:   "test/repo",
		Prompt: "test prompt",
		Status: models.TaskStatusQueued,
	}
	if err := DB.Create(task).Error; err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	// Verify record exists
	var count int64
	DB.Model(&models.Task{}).Count(&count)
	if count != 1 {
		t.Fatal("Test record was not created")
	}

	// Reset database
	err := ResetDatabase()
	if err != nil {
		t.Fatalf("ResetDatabase() failed: %v", err)
	}

	// Verify record is gone
	DB.Model(&models.Task{}).Count(&count)
	if count != 0 {
		t.Fatal("ResetDatabase() did not clear data")
	}

	// Verify table still exists
	var tableCount int64
	err = DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tasks'").Scan(&tableCount).Error
	if err != nil {
		t.Fatalf("Failed to check for tasks table: %v", err)
	}
	if tableCount != 1 {
		t.Fatal("ResetDatabase() removed table structure")
	}
}

func TestTaskCRUDOperations(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Setup database
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	if err := Migrate(); err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}

	// Test Create
	task := &models.Task{
		ID:       "test-task-id",
		Repo:     "github.com/test/repo",
		Branch:   "amp/test123",
		ThreadID: "thread-123",
		Prompt:   "Implement feature X",
		Status:   models.TaskStatusQueued,
		Attempts: 0,
	}

	err := DB.Create(task).Error
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Test Read
	var retrievedTask models.Task
	err = DB.First(&retrievedTask, "id = ?", task.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Retrieved task ID = %v, want %v", retrievedTask.ID, task.ID)
	}
	if retrievedTask.Repo != task.Repo {
		t.Errorf("Retrieved task Repo = %v, want %v", retrievedTask.Repo, task.Repo)
	}
	if retrievedTask.Status != task.Status {
		t.Errorf("Retrieved task Status = %v, want %v", retrievedTask.Status, task.Status)
	}

	// Test Update
	retrievedTask.Status = models.TaskStatusRunning
	retrievedTask.Attempts = 1
	err = DB.Save(&retrievedTask).Error
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Verify update
	var updatedTask models.Task
	err = DB.First(&updatedTask, "id = ?", task.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve updated task: %v", err)
	}

	if updatedTask.Status != models.TaskStatusRunning {
		t.Errorf("Updated task Status = %v, want %v", updatedTask.Status, models.TaskStatusRunning)
	}
	if updatedTask.Attempts != 1 {
		t.Errorf("Updated task Attempts = %v, want %v", updatedTask.Attempts, 1)
	}

	// Test timestamps are updated
	if !updatedTask.UpdatedAt.After(updatedTask.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}

	// Test Delete
	err = DB.Delete(&updatedTask).Error
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify deletion
	var count int64
	DB.Model(&models.Task{}).Where("id = ?", task.ID).Count(&count)
	if count != 0 {
		t.Error("Task was not deleted")
	}
}

func TestTaskQueries(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Setup database
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	if err := Migrate(); err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}

	// Create test data
	tasks := []*models.Task{
		{
			ID:     "task-1",
			Repo:   "github.com/test/repo1",
			Status: models.TaskStatusQueued,
		},
		{
			ID:     "task-2", 
			Repo:   "github.com/test/repo1",
			Status: models.TaskStatusRunning,
		},
		{
			ID:     "task-3",
			Repo:   "github.com/test/repo2",
			Status: models.TaskStatusSuccess,
		},
	}

	for _, task := range tasks {
		if err := DB.Create(task).Error; err != nil {
			t.Fatalf("Failed to create test task: %v", err)
		}
	}

	// Test query by status
	var queuedTasks []models.Task
	err := DB.Where("status = ?", models.TaskStatusQueued).Find(&queuedTasks).Error
	if err != nil {
		t.Fatalf("Failed to query by status: %v", err)
	}
	if len(queuedTasks) != 1 {
		t.Errorf("Query by status returned %d tasks, want 1", len(queuedTasks))
	}

	// Test query by repo
	var repo1Tasks []models.Task
	err = DB.Where("repo = ?", "github.com/test/repo1").Find(&repo1Tasks).Error
	if err != nil {
		t.Fatalf("Failed to query by repo: %v", err)
	}
	if len(repo1Tasks) != 2 {
		t.Errorf("Query by repo returned %d tasks, want 2", len(repo1Tasks))
	}

	// Test ordering by created_at
	var allTasks []models.Task
	err = DB.Order("created_at DESC").Find(&allTasks).Error
	if err != nil {
		t.Fatalf("Failed to query with ordering: %v", err)
	}
	if len(allTasks) != 3 {
		t.Errorf("Query returned %d tasks, want 3", len(allTasks))
	}

	// Verify ordering (newest first)
	for i := 1; i < len(allTasks); i++ {
		if allTasks[i].CreatedAt.After(allTasks[i-1].CreatedAt) {
			t.Error("Tasks are not ordered by created_at DESC")
		}
	}
}

func TestDatabaseConcurrency(t *testing.T) {
	dbPath := setupTestDB(t)
	defer teardownTestDB(t, dbPath)

	// Setup database
	if err := Connect(dbPath); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	if err := Migrate(); err != nil {
		t.Fatalf("Migrate() failed: %v", err)
	}

	// Test concurrent writes
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			task := &models.Task{
				ID:     string(rune('a' + id)),
				Repo:   "github.com/test/concurrent",
				Status: models.TaskStatusQueued,
			}
			err := DB.Create(task).Error
			if err != nil {
				t.Errorf("Concurrent create failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent test timed out")
		}
	}

	// Verify all records were created
	var count int64
	DB.Model(&models.Task{}).Where("repo = ?", "github.com/test/concurrent").Count(&count)
	if count != int64(numGoroutines) {
		t.Errorf("Concurrent writes created %d records, want %d", count, numGoroutines)
	}
}
