/**
 * 数据库 Schema 定义入口
 * 表结构见 schema_tables.go，索引见 schema_indexes.go
 */
package persistence

import (
	"database/sql"
)

// Schema 完整的数据库 Schema（表结构 + 索引）
const Schema = SchemaTables + SchemaIndexes

// InitSchema 初始化数据库 Schema（表结构 + 索引）
func InitSchema(db *sql.DB) error {
	if _, err := db.Exec(Schema); err != nil {
		return err
	}
	return nil
}
