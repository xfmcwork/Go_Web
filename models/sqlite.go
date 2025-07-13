package models

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var (
	db       *sql.DB
	onceInit sync.Once
)

func Init(dbPath string) error {
	var err error
	onceInit.Do(func() {
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Println("打开数据库失败: %w", err)
		}
		if err = db.Ping(); err != nil {
			log.Println("连接数据库失败: %w", err)
		}
		if err = createTable(); err != nil {
			log.Println("创建表失败: %w", err)
		}

		log.Println("数据库读取成功")
	})

	return err
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS user (
	UID INTEGER PRIMARY KEY AUTOINCREMENT,
    Name TEXT NOT NULL UNIQUE,
	Email TEXT,
    Password TEXT NOT NULL,
	Time TEXT,
	RToken TEXT,
	RTExpiration TEXT,
    Token TEXT
	);
	`

	_, err := db.Exec(query)
	return err
}

func Query(table string, conditions map[string]interface{}) ([]map[string]interface{}, error) {
	if table == "" {
		log.Println("表名不能为空")
	}

	var (
		whereClauses []string
		args         []interface{}
	)

	for key, value := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	whereStr := ""
	if len(whereClauses) > 0 {
		whereStr = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("SELECT * FROM %s %s", table, whereStr)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("查询失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Println("获取列名失败: %w", err)
	}

	results := []map[string]interface{}{}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Println("扫描行失败: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			row[col] = v
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		log.Println("遍历结果集失败: %w", err)
	}

	return results, nil
}

// 通用插入函数
func Insert(table string, data map[string]interface{}) (int64, error) {
	if table == "" {
		log.Println("表名不能为空")
	}
	if len(data) == 0 {
		log.Println("插入数据不能为空")
	}

	var (
		columns      []string
		placeholders []string
		values       []interface{}
	)

	for column, value := range data {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	columnStr := strings.Join(columns, ", ")
	placeholderStr := strings.Join(placeholders, ", ")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, columnStr, placeholderStr)

	result, err := db.Exec(query, values...)
	if err != nil {
		log.Println("插入失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println("获取插入ID失败: %w", err)
	}

	return id, nil
}

// 通用更新函数
func Update(table string, data map[string]interface{}, conditions map[string]interface{}) (int64, error) {
	if table == "" {
		log.Println("表名不能为空")
	}
	if len(data) == 0 {
		log.Println("更新数据不能为空")
	}

	var (
		setClauses   []string
		whereClauses []string
		args         []interface{}
	)

	for column, value := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", column))
		args = append(args, value)
	}

	for key, value := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	whereStr := ""
	if len(whereClauses) > 0 {
		whereStr = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	setStr := strings.Join(setClauses, ", ")
	query := fmt.Sprintf("UPDATE %s SET %s %s", table, setStr, whereStr)

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Println("更新失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("获取更新行数失败: %w", err)
	}

	return rowsAffected, nil
}

// 通用删除函数
func Delete(table string, conditions map[string]interface{}) (int64, error) {
	if table == "" {
		log.Println("表名不能为空")
	}

	var (
		whereClauses []string
		args         []interface{}
	)

	for key, value := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	whereStr := ""
	if len(whereClauses) > 0 {
		whereStr = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("DELETE FROM %s %s", table, whereStr)

	result, err := db.Exec(query, args...)
	if err != nil {
		log.Println("删除失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("获取删除行数失败: %w", err)
	}

	return rowsAffected, nil
}
