package db

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
)

// KeyDatabase 是数据库实例列表的配置键: 列表为空则不加载任何实例.
const KeyDatabase = "database"

const (
	fieldName              = "name"
	fieldDriver            = "driver"
	fieldDSN               = "dsn"
	fieldNet               = "net"
	fieldHost              = "host"
	fieldPort              = "port"
	fieldUser              = "user"
	fieldPassword          = "password"
	fieldDatabase          = "database"
	fieldCharset           = "charset"
	fieldCollation         = "collation"
	fieldParseTime         = "parseTime"
	fieldLoc               = "loc"
	fieldTimeout           = "timeout"
	fieldReadTimeout       = "readTimeout"
	fieldWriteTimeout      = "writeTimeout"
	fieldTLS               = "tls"
	fieldMultiStatements   = "multiStatements"
	fieldInterpolateParams = "interpolateParams"
	fieldParams            = "params"

	defaultMySQLNet  = "tcp"
	defaultMySQLPort = 3306
	defaultParseTime = true
)

// mysqlNets 是 go-sql-driver/mysql 支持的连接协议.
var mysqlNets = map[string]struct{}{
	"tcp":  {},
	"tcp4": {},
	"tcp6": {},
	"unix": {},
}

// Load 解析 database 列表; 同名实例只取第一个, 后续同名项跳过.
func Load(cfg context2.ConfigContext) ([]Config, error) {
	raws := config.Maps(cfg, KeyDatabase)
	out := make([]Config, 0, len(raws))
	seen := make(map[string]struct{}, len(raws))

	for _, raw := range raws {
		f := fields(raw)

		name := strings.TrimSpace(f.str(fieldName, ""))
		if name == "" {
			return nil, invalidInstance("", "", fieldName)
		}
		if _, dup := seen[name]; dup {
			continue
		}

		driver := f.str(fieldDriver, "")
		parse, ok := instanceParsers[driver]
		if !ok {
			return nil, errs.New(ErrUnsupported).With("name", name, "driver", driver)
		}
		inst, err := parse(name, f)
		if err != nil {
			return nil, err
		}

		seen[name] = struct{}{}
		out = append(out, inst)
	}
	return out, nil
}

// Config 是一个数据库实例的最终配置; Driver 始终是 database/sql 的注册名.
type Config struct {
	Name   string
	Driver string
	DSN    string
}

// instanceParser 按配置中的 driver 解析各自特有的字段.
type instanceParser func(name string, f fields) (Config, error)

var instanceParsers = map[string]instanceParser{
	DriverSQLite:  parseSQLite,
	DriverMySQL:   parseMySQL,
	DriverMariaDB: parseMySQL,
}

func parseSQLite(name string, f fields) (Config, error) {
	dsn := f.str(fieldDSN, "")
	if dsn == "" {
		return Config{}, invalidInstance(name, DriverSQLite, fieldDSN)
	}
	return Config{Name: name, Driver: DriverSQLite, DSN: dsn}, nil
}

// parseMySQL 依据连接属性拼装 DSN, 同时服务于 mysql 与 mariadb.
func parseMySQL(name string, f fields) (Config, error) {
	host, user, dbName := f.str(fieldHost, ""), f.str(fieldUser, ""), f.str(fieldDatabase, "")
	if host == "" || user == "" || dbName == "" {
		return Config{}, invalidInstance(name, DriverMySQL, "host|user|database")
	}

	network := f.str(fieldNet, defaultMySQLNet)
	if _, ok := mysqlNets[network]; !ok {
		return Config{}, invalidInstance(name, DriverMySQL, fieldNet)
	}

	dsn := mysql.NewConfig()
	dsn.Net = network
	dsn.Addr = net.JoinHostPort(host, strconv.Itoa(f.num(fieldPort, defaultMySQLPort)))
	if network == "unix" {
		dsn.Addr = host
	}
	dsn.User = user
	dsn.Passwd = f.str(fieldPassword, "")
	dsn.DBName = dbName
	dsn.ParseTime = f.boolean(fieldParseTime, defaultParseTime)
	dsn.Collation = f.str(fieldCollation, dsn.Collation)
	dsn.TLSConfig = f.str(fieldTLS, "")
	dsn.MultiStatements = f.boolean(fieldMultiStatements, false)
	dsn.InterpolateParams = f.boolean(fieldInterpolateParams, false)
	dsn.Params = f.stringMap(fieldParams)

	if loc := f.str(fieldLoc, ""); loc != "" {
		parsed, err := time.LoadLocation(loc)
		if err != nil {
			return Config{}, invalidInstance(name, DriverMySQL, fieldLoc)
		}
		dsn.Loc = parsed
	}

	timeouts := []struct {
		field string
		dst   *time.Duration
	}{
		{fieldTimeout, &dsn.Timeout},
		{fieldReadTimeout, &dsn.ReadTimeout},
		{fieldWriteTimeout, &dsn.WriteTimeout},
	}
	for _, tt := range timeouts {
		d, ok := f.duration(tt.field)
		if !ok {
			return Config{}, invalidInstance(name, DriverMySQL, tt.field)
		}
		*tt.dst = d
	}

	if charset := f.str(fieldCharset, ""); charset != "" {
		if dsn.Params == nil {
			dsn.Params = make(map[string]string, 1)
		}
		dsn.Params[fieldCharset] = charset
	}

	return Config{Name: name, Driver: DriverMySQL, DSN: dsn.FormatDSN()}, nil
}

func invalidInstance(name, driver, field string) error {
	return errs.New(ErrInstanceConfig).With("name", name, "driver", driver, "field", field)
}

// fields 包装原始配置对象, 提供带默认值的读取.
type fields map[string]any

func (f fields) str(key, def string) string {
	v, ok := f[key].(string)
	if !ok {
		return def
	}
	return v
}

func (f fields) num(key string, def int) int {
	switch v := f[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return def
	}
}

func (f fields) boolean(key string, def bool) bool {
	v, ok := f[key].(bool)
	if !ok {
		return def
	}
	return v
}

// duration 解析 "5s"/"1m" 形式的时长; 缺省视为合法(0), 非法返回 false.
func (f fields) duration(key string) (time.Duration, bool) {
	raw := f.str(key, "")
	if raw == "" {
		return 0, true
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, false
	}
	return d, true
}

func (f fields) stringMap(key string) map[string]string {
	raw, ok := f[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out
}
