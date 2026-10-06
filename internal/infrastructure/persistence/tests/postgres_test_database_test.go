package persistence_test

import (
	"os"
	"strings"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testDatabaseSuffix = "_test"

func openTestDatabase(t *testing.T) *gorm.DB {
	databaseUrl := os.Getenv("TEST_POSTGRES_DSN")
	if databaseUrl == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	connectionConfig, err := pgconn.ParseConfig(databaseUrl)
	require.NoError(t, err)
	// guards against wiping a real database: the suite drops tables
	require.True(t, strings.HasSuffix(connectionConfig.Database, testDatabaseSuffix), "TEST_POSTGRES_DSN must name a database ending with _test")
	database, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.Migrator().DropTable(&entities.ApiKey{}))
	require.NoError(t, database.AutoMigrate(&entities.ApiKey{}))
	return database
}
