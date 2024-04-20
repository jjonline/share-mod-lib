package migrate

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"github.com/logrusorgru/aurora/v3"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 迁移操作
const (
	MigrationUp   uint8 = 1
	MigrationDown uint8 = 2
)

// Migration 解析后的迁移文件数据结构
type Migration struct {
	Id   string
	Up   []string
	Down []string

	DisableTransactionUp   bool
	DisableTransactionDown bool
}

// MigrationModel MigrationModel
type MigrationModel struct {
	ID        uint64    `json:"id"`
	Migration string    `json:"migration"`
	CreatedAt time.Time `json:"created_at"`
}

// MigrationOutput 列表输出
type MigrationOutput struct {
	Migration string
	Status    string
}

// Config 配置
type Config struct {
	Dir       string    // migrate迁移文件的路径，相对路径<相对main入口|编译后二进制文件>
	Fs        *embed.FS // `Dir`参数指向的路径里的embed机制形成的fs.FS变量，以实现build编译嵌入embed迁移文件
	TableName string    // 记录migrate迁移历史的数据库表名
	DB        *sql.DB   // 数据库驱动句柄，用于操纵数据库
}

// Migrate Migrate对象
type Migrate struct {
	config Config
}

// New 实例化migrate迁移对象
func New(conf Config) *Migrate {
	// Dir 和 Fs不得同时都为空，建议是两者都传，方便go build嵌入迁移文件
	if conf.TableName == "" || conf.DB == nil || (conf.Dir == "" && conf.Fs == nil) {
		panic("NewMigrate初始化参数错误")
	}

	m := &Migrate{conf}
	if err := m.InitMigrationTable(); err != nil {
		panic("Migrate数据迁移记录表创建失败")
	}
	return m
}

// InitMigrationTable 初始化数据迁移表
func (m Migrate) InitMigrationTable() (err error) {
	query := `CREATE TABLE IF NOT EXISTS @TableName (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,
  migration varchar(200) NOT NULL,
  created_at datetime NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 comment='数据迁移记录表';`
	_, err = m.config.DB.Exec(strings.Replace(query, "@TableName", m.config.TableName, 1))
	return
}

// Status 查看迁移文件列表状态
func (m Migrate) Status() (err error) {
	migrations, err := m.FindMigrations()
	if err != nil {
		return
	}

	// 已执行的迁移文件
	executedRecords, err := m.GetExecutedMigrations()
	if err != nil {
		return
	}
	executedRecordsMap := make(map[string]MigrationModel)
	for i := range executedRecords {
		executedRecordsMap[executedRecords[i].Migration] = executedRecords[i]
	}

	//输出
	var output []MigrationOutput
	for i := range migrations {
		item := MigrationOutput{
			Migration: migrations[i].Id,
			Status:    "No",
		}
		filename := migrations[i].Id
		if record, ok := executedRecordsMap[filename]; ok {
			item.Status = record.CreatedAt.Format("2006-01-02 15:04:05")
		}
		output = append(output, item)
	}
	Output(output, GridASCII)
	return
}

// Create 创建迁移文件--仅开发阶段可用
func (m Migrate) Create(filename string) (err error) {
	if filename == "" {
		fmt.Println(aurora.Bold(aurora.Red("migration error:")), "请输入要创建的迁移文件名称")
		return
	}

	content := `-- +migrate Up


-- +migrate Down


`
	filename = time.Now().Format("20060102150405") + "-" + filename + ".sql"
	fullPath := m.config.Dir + "/" + filename
	if CheckFileExist(fullPath) {
		errText := fmt.Sprintf("migration create failure: %s exists", fullPath)
		fmt.Println(errText)
		return fmt.Errorf(errText)
	}
	_, err = WriteFile(fullPath, content)
	if err != nil {
		errText := "migration create failure: " + err.Error()
		fmt.Println(errText)
		return fmt.Errorf(errText)
	}
	fmt.Println(aurora.Bold(aurora.Green("migration create success:")), filename)
	return
}

// GetExecutedMigrations 从Db获取已执行的迁移记录
func (m Migrate) GetExecutedMigrations() (migrations []MigrationModel, err error) {
	migrations = make([]MigrationModel, 0)
	query := "select id,migration,created_at from %s order by id asc"
	rows, err := m.config.DB.Query(fmt.Sprintf(query, m.config.TableName))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		item := MigrationModel{}
		if err = rows.Scan(&item.ID, &item.Migration, &item.CreatedAt); err != nil {
			return
		}
		migrations = append(migrations, item)
	}
	return
}

// ExecUp 执行迁移文件
func (m Migrate) ExecUp() (err error) {
	migrations, err := m.FindMigrations()
	if err != nil {
		return
	}
	if len(migrations) == 0 {
		fmt.Println(aurora.Bold(aurora.Red("migrate error:")), "无迁移文件")
		return
	}

	// 已执行的迁移文件
	executedRecords, err := m.GetExecutedMigrations()
	if err != nil {
		return
	}

	executedRecordsMap := make(map[string]MigrationModel)
	for i := range executedRecords {
		executedRecordsMap[executedRecords[i].Migration] = executedRecords[i]
	}

	for i := range migrations {
		filename := migrations[i].Id
		if _, ok := executedRecordsMap[filename]; ok {
			continue
		}

		fmt.Println(aurora.Bold(aurora.Green("migrate-up:")), filename)

		err = m.execute(filename, MigrationUp, migrations[i].Up)
		if err != nil {
			fmt.Println(aurora.Bold(aurora.Red("migrate-up error:")), err.Error())
			return nil
		}
		fmt.Println(aurora.Bold(aurora.Green("migrate-up success:")), filename)
	}

	fmt.Println(aurora.Bold(aurora.Green("migrate finish.")))
	return
}

// ExecDown 回滚已执行的迁移文件，每次回滚一个
func (m Migrate) ExecDown(filename string) (err error) {
	migrations, err := m.FindMigrations()
	if err != nil {
		return
	}

	// 已执行的迁移文件
	executedRecords, err := m.GetExecutedMigrations()
	if err != nil {
		return
	}
	count := len(executedRecords)
	if count == 0 {
		fmt.Println(aurora.Bold(aurora.Red("migrate error:")), "没有已执行的迁移记录，无法执行回滚操作")
		return
	}

	var lastMigration MigrationModel

	//回滚指定迁移文件
	if filename != "" {
		for i := range executedRecords {
			if executedRecords[i].Migration == filename {
				lastMigration = executedRecords[i]
				break
			}
		}
		if lastMigration.Migration == "" {
			fmt.Printf("%v 无法执行回滚操作,未找到迁移文件%s\n",
				aurora.Bold(aurora.Red("migrate error:")), lastMigration.Migration)
			return
		}
	} else {
		lastMigration = executedRecords[count-1]
	}

	for i := range migrations {
		if migrations[i].Id == lastMigration.Migration {
			fmt.Println(aurora.Bold(aurora.Green("migrate-down:")), lastMigration.Migration)
			err = m.execute(lastMigration.Migration, MigrationDown, migrations[i].Down)
			if err != nil {
				fmt.Println(aurora.Bold(aurora.Red("migrate-down error:")), err.Error())
			}
			fmt.Println(aurora.Bold(aurora.Green("migrate-down success:")), lastMigration.Migration)
			return
		}
	}

	//未找到对应的迁移文件
	fmt.Printf("%v 无法执行回滚操作,未找到迁移文件%s\n",
		aurora.Bold(aurora.Red("migrate error:")), lastMigration.Migration)
	return
}

// execute 执行sql操作
func (m Migrate) execute(filename string, action uint8, queries []string) (err error) {
	if action == MigrationUp && len(queries) == 0 {
		err = fmt.Errorf("迁移文件 %s 无数据", filename)
		return
	}

	tx, _ := m.config.DB.Begin()
	for _, query := range queries {
		if _, err = tx.Exec(query); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if action == MigrationUp {
		query := "insert into %s (migration, created_at) values ('%s', '%s');"
		if _, err = tx.Exec(fmt.Sprintf(query, m.config.TableName, filename, time.Now().Format("2006-01-02 15:04:15"))); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if action == MigrationDown {
		query := "delete from %s where migration = '%s';"
		if _, err = tx.Exec(fmt.Sprintf(query, m.config.TableName, filename)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// FindMigrations 从本地文件系统|fs.FS读取全部迁移文件
func (m Migrate) FindMigrations() ([]*Migration, error) {
	migrations := make([]*Migration, 0)

	// ① 优先从 fs.FS 文件系统获取迁移文件内容
	if m.config.Fs != nil {
		dirEntry, err := m.config.Fs.ReadDir(".")
		if err != nil {
			return nil, err
		}
		for _, one := range dirEntry {
			if !one.IsDir() && strings.HasSuffix(one.Name(), ".sql") {
				if content, err := m.config.Fs.ReadFile(one.Name()); err == nil {
					if migration, err := parseMigration(one.Name(), bytes.NewReader(content)); err == nil {
						migrations = append(migrations, migration)
					}
				}
			}
		}

		// 排序
		sort.Sort(byId(migrations))

		return migrations, nil
	}

	// ② 降级从本地文件系统获取迁移文件内容
	filesystem := http.Dir(m.config.Dir)
	file, err := filesystem.Open("/")
	if err != nil {
		return nil, err
	}

	files, err := file.Readdir(0)
	if err != nil {
		return nil, err
	}

	for _, info := range files {
		if strings.HasSuffix(info.Name(), ".sql") {
			migration, err := migrationFromFile(filesystem, info)
			if err != nil {
				return nil, err
			}

			migrations = append(migrations, migration)
		}
	}

	// Make sure migrations are sorted
	sort.Sort(byId(migrations))

	return migrations, nil
}

// migrationFromFile 从本地文件系统获取迁移文件内容
func migrationFromFile(dir http.FileSystem, info os.FileInfo) (*Migration, error) {
	path := fmt.Sprintf("/%s", strings.TrimPrefix(info.Name(), "/"))
	file, err := dir.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error while opening %s: %s", info.Name(), err)
	}
	defer func() { _ = file.Close() }()

	migration, err := parseMigration(info.Name(), file)
	if err != nil {
		return nil, fmt.Errorf("error while parsing %s: %s", info.Name(), err)
	}
	return migration, nil
}

// parseMigration 解析迁移文件
func parseMigration(id string, r io.ReadSeeker) (*Migration, error) {
	m := &Migration{
		Id: id,
	}

	parsed, err := ParseMigration(r)
	if err != nil {
		return nil, fmt.Errorf("error parsing migration (%s): %s", id, err)
	}

	m.Up = parsed.UpStatements
	m.Down = parsed.DownStatements

	m.DisableTransactionUp = parsed.DisableTransactionUp
	m.DisableTransactionDown = parsed.DisableTransactionDown

	return m, nil
}

// byId
type byId []*Migration

func (b byId) Len() int           { return len(b) }
func (b byId) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byId) Less(i, j int) bool { return b[i].Less(b[j]) }

var numberPrefixRegex = regexp.MustCompile(`^(\d+).*$`)

// Less less
func (m Migration) Less(other *Migration) bool {
	switch {
	case m.isNumeric() && other.isNumeric() && m.VersionInt() != other.VersionInt():
		return m.VersionInt() < other.VersionInt()
	case m.isNumeric() && !other.isNumeric():
		return true
	case !m.isNumeric() && other.isNumeric():
		return false
	default:
		return m.Id < other.Id
	}
}

// isNumeric isNumeric
func (m Migration) isNumeric() bool {
	return len(m.NumberPrefixMatches()) > 0
}

// NumberPrefixMatches 匹配数字版本号
func (m Migration) NumberPrefixMatches() []string {
	return numberPrefixRegex.FindStringSubmatch(m.Id)
}

// VersionInt VersionInt
func (m Migration) VersionInt() int64 {
	v := m.NumberPrefixMatches()[1]
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Could not parse %q into int64: %s", v, err))
	}
	return value
}
