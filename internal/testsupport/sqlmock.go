package testsupport

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMockGormDB 创建一个基于 sqlmock 的 GORM 连接，便于为 repository/service 编写最小单测。
func NewMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = rawDB.Close()
	}

	return db, mock, cleanup
}

// NewRows 是对 sqlmock.NewRows 的薄封装，避免测试文件重复导入。
func NewRows(columns []string) *sqlmock.Rows {
	return sqlmock.NewRows(columns)
}

// SQLResult 返回一个常用的 Exec 成功结果。
func SQLResult(lastInsertID int64, rowsAffected int64) sql.Result {
	return sqlmock.NewResult(lastInsertID, rowsAffected)
}
