package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/persistence"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// the shape the analysis_results table had before grade records replaced it
type analysisResultRow struct {
	ID              uint                        `gorm:"primaryKey"`
	AnalysisEventID uint                        `gorm:"not null;uniqueIndex"`
	Symbol          string                      `gorm:"size:32;not null"`
	Category        string                      `gorm:"size:16;not null"`
	SymbolName      string                      `gorm:"size:100;not null;default:''"`
	Grade           string                      `gorm:"size:16;not null"`
	Confidence      int                         `gorm:"not null"`
	TimeHorizon     string                      `gorm:"size:16;not null"`
	Reason          string                      `gorm:"type:text;not null"`
	KeyEvents       []entities.AnalysisKeyEvent `gorm:"type:jsonb;serializer:json;not null"`
	RiskFactors     []string                    `gorm:"type:jsonb;serializer:json;not null"`
	Evidence        []entities.AnalysisEvidence `gorm:"type:jsonb;serializer:json;not null"`
	Price           decimal.NullDecimal         `gorm:"type:numeric"`
	PriceCurrency   string                      `gorm:"size:8"`
	PricedAt        *time.Time
	PriceSource     string `gorm:"size:32"`
	CreatedAt       time.Time
}

func (analysisResultRow) TableName() string {
	return "analysis_results"
}

// the shape the combined_grades table had before background grade records replaced it
type combinedGradeRow struct {
	ID            uint    `gorm:"primaryKey"`
	Symbol        string  `gorm:"size:32;not null"`
	Category      string  `gorm:"size:16;not null"`
	TradingDay    string  `gorm:"size:10;not null"`
	Grade         string  `gorm:"size:16;not null"`
	CombinedScore float64 `gorm:"not null"`
	Confidence    int     `gorm:"not null"`
	UpdatedAt     time.Time
}

func (combinedGradeRow) TableName() string {
	return "combined_grades"
}

func openDatabaseWithLegacyTables(t *testing.T) *gorm.DB {
	database := openTestDatabase(t)
	require.NoError(t, database.Migrator().DropTable(&analysisResultRow{}, &combinedGradeRow{}))
	require.NoError(t, database.AutoMigrate(&analysisResultRow{}, &combinedGradeRow{}))
	return database
}

func TestGradeRecordRepository_ImportLegacyGradesMovesAnalysisResultsAsManualRecords(t *testing.T) {
	database := openDatabaseWithLegacyTables(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 10, 6, 16, 40, 14, 0, time.UTC)
	pricedAt := createdAt.Add(-time.Minute)
	keyEvents := []entities.AnalysisKeyEvent{{Title: "法說會", Link: "https://news/1", PublishedAt: createdAt.Add(-time.Hour)}}
	evidence := []entities.AnalysisEvidence{{Title: "法說會", Link: "https://news/1", PublishedAt: createdAt.Add(-time.Hour), ProviderName: "鉅亨網"}}
	for analysisEventID := uint(1); analysisEventID <= 9; analysisEventID++ {
		require.NoError(t, database.Create(&analysisResultRow{
			AnalysisEventID: analysisEventID, Symbol: "2330", Category: "twStock", SymbolName: "台積電", Grade: "bullish", Confidence: 70, TimeHorizon: "mid", Reason: "理由",
			KeyEvents: keyEvents, RiskFactors: []string{"匯率"}, Evidence: evidence,
			Price: decimal.NewNullDecimal(decimal.RequireFromString("1025.5")), PriceCurrency: "TWD", PricedAt: &pricedAt, PriceSource: "TWSE", CreatedAt: createdAt.Add(time.Duration(analysisEventID) * time.Minute),
		}).Error)
	}

	require.NoError(t, persistence.NewGradeRecordRepository(database).ImportLegacyGrades(ctx))

	gradeRecords := []entities.GradeRecord{}
	require.NoError(t, database.Find(&gradeRecords).Error)
	require.Len(t, gradeRecords, 9)
	movedRecord, err := persistence.NewGradeRecordRepository(database).FindByAnalysisEventID(ctx, 9)
	require.NoError(t, err)
	require.NotNil(t, movedRecord)
	assert.Equal(t, "manual", movedRecord.Source)
	assert.Equal(t, "bullish", movedRecord.Grade)
	assert.Equal(t, 70, movedRecord.Confidence)
	assert.Equal(t, "mid", movedRecord.TimeHorizon)
	assert.Equal(t, "理由", movedRecord.Reason)
	assert.Equal(t, "台積電", movedRecord.SymbolName)
	assert.Equal(t, keyEvents[0].Link, movedRecord.KeyEvents[0].Link)
	assert.Equal(t, []string{"匯率"}, movedRecord.RiskFactors)
	assert.Equal(t, "鉅亨網", movedRecord.Evidence[0].ProviderName)
	assert.Equal(t, "1025.5", movedRecord.Price.Decimal.String())
	assert.Equal(t, "TWD", movedRecord.PriceCurrency)
	assert.True(t, pricedAt.Equal(*movedRecord.PricedAt))
	assert.Equal(t, "TWSE", movedRecord.PriceSource)
	assert.True(t, createdAt.Add(9*time.Minute).Equal(movedRecord.CreatedAt))
	assert.Empty(t, movedRecord.TradingDay)
	assert.Nil(t, movedRecord.CombinedScore)
}

func TestGradeRecordRepository_ImportLegacyGradesMovesCombinedGradesAsBackgroundRecords(t *testing.T) {
	database := openDatabaseWithLegacyTables(t)
	updatedAt := time.Date(2026, 10, 7, 20, 10, 0, 0, backtestTaipei)
	require.NoError(t, database.Create(&combinedGradeRow{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "neutral", CombinedScore: 0.2, Confidence: 52, UpdatedAt: updatedAt}).Error)

	require.NoError(t, persistence.NewGradeRecordRepository(database).ImportLegacyGrades(context.Background()))

	gradeRecords := []entities.GradeRecord{}
	require.NoError(t, database.Find(&gradeRecords).Error)
	require.Len(t, gradeRecords, 1)
	assert.Equal(t, "background", gradeRecords[0].Source)
	assert.Equal(t, "2330", gradeRecords[0].Symbol)
	assert.Equal(t, "2026-10-07", gradeRecords[0].TradingDay)
	assert.Equal(t, "neutral", gradeRecords[0].Grade)
	assert.Equal(t, 52, gradeRecords[0].Confidence)
	assert.Equal(t, 0.2, *gradeRecords[0].CombinedScore)
	assert.Equal(t, "short", gradeRecords[0].TimeHorizon)
	assert.True(t, updatedAt.Equal(gradeRecords[0].CreatedAt))
	assert.Nil(t, gradeRecords[0].AnalysisEventID)
	assert.Empty(t, gradeRecords[0].SymbolName)
	assert.False(t, gradeRecords[0].Price.Valid)
}

func TestGradeRecordRepository_ImportLegacyGradesRunsOnlyOnce(t *testing.T) {
	database := openDatabaseWithLegacyTables(t)
	require.NoError(t, database.Create(&analysisResultRow{AnalysisEventID: 1, Symbol: "BTC", Category: "crypto", Grade: "neutral", TimeHorizon: "short", Reason: "r", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}}).Error)
	require.NoError(t, database.Create(&combinedGradeRow{Symbol: "2330", Category: "twStock", TradingDay: "2026-10-07", Grade: "bullish"}).Error)
	gradeRecordRepository := persistence.NewGradeRecordRepository(database)

	require.NoError(t, gradeRecordRepository.ImportLegacyGrades(context.Background()))
	require.NoError(t, gradeRecordRepository.ImportLegacyGrades(context.Background()))

	gradeRecordCount := int64(0)
	require.NoError(t, database.Model(&entities.GradeRecord{}).Count(&gradeRecordCount).Error)
	assert.Equal(t, int64(2), gradeRecordCount)
	assert.False(t, database.Migrator().HasTable("analysis_results"))
	assert.False(t, database.Migrator().HasTable("combined_grades"))
}

func TestGradeRecordRepository_ImportLegacyGradesWithoutLegacyTablesChangesNothing(t *testing.T) {
	database := openTestDatabase(t)
	require.NoError(t, database.Migrator().DropTable(&analysisResultRow{}, &combinedGradeRow{}))

	require.NoError(t, persistence.NewGradeRecordRepository(database).ImportLegacyGrades(context.Background()))

	gradeRecordCount := int64(0)
	require.NoError(t, database.Model(&entities.GradeRecord{}).Count(&gradeRecordCount).Error)
	assert.Equal(t, int64(0), gradeRecordCount)
}

func TestGradeRecordRepository_ImportLegacyGradesKeepsTheLegacyRowsWhenMovingFails(t *testing.T) {
	database := openDatabaseWithLegacyTables(t)
	require.NoError(t, database.Create(&analysisResultRow{AnalysisEventID: 1, Symbol: "BTC", Category: "crypto", Grade: "neutral", TimeHorizon: "short", Reason: "r", KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{}}).Error)
	// the analysis already has a grade record, so moving its result breaks the one-record-per-analysis rule
	require.NoError(t, persistence.NewGradeRecordRepository(database).Create(context.Background(), &entities.GradeRecord{
		Source: "manual", AnalysisEventID: new(uint(1)), Symbol: "BTC", Category: "crypto", Grade: "bullish", TimeHorizon: "short",
		KeyEvents: []entities.AnalysisKeyEvent{}, RiskFactors: []string{}, Evidence: []entities.AnalysisEvidence{},
	}))

	importError := persistence.NewGradeRecordRepository(database).ImportLegacyGrades(context.Background())

	assert.Error(t, importError)
	assert.True(t, database.Migrator().HasTable("analysis_results"))
	legacyRowCount := int64(0)
	require.NoError(t, database.Model(&analysisResultRow{}).Count(&legacyRowCount).Error)
	assert.Equal(t, int64(1), legacyRowCount)
}

type brokenAnalysisResultRow struct {
	ID uint `gorm:"primaryKey"`
}

func (brokenAnalysisResultRow) TableName() string {
	return "analysis_results"
}

type brokenCombinedGradeRow struct {
	ID uint `gorm:"primaryKey"`
}

func (brokenCombinedGradeRow) TableName() string {
	return "combined_grades"
}

func TestGradeRecordRepository_ImportLegacyGradesFailsWhenALegacyTableCannotBeRead(t *testing.T) {
	testCases := []struct {
		name              string
		legacyTableName   string
		createLegacyTable func(database *gorm.DB) error
	}{
		{name: "analysis results", legacyTableName: "analysis_results", createLegacyTable: func(database *gorm.DB) error { return database.AutoMigrate(&brokenAnalysisResultRow{}) }},
		{name: "combined grades", legacyTableName: "combined_grades", createLegacyTable: func(database *gorm.DB) error { return database.AutoMigrate(&brokenCombinedGradeRow{}) }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database := openTestDatabase(t)
			require.NoError(t, database.Migrator().DropTable(&analysisResultRow{}, &combinedGradeRow{}))
			require.NoError(t, testCase.createLegacyTable(database))

			importError := persistence.NewGradeRecordRepository(database).ImportLegacyGrades(context.Background())

			assert.Error(t, importError)
			assert.True(t, database.Migrator().HasTable(testCase.legacyTableName))
		})
	}
}
