package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const ctxLoggerKey = "zapLogger"

type Logger struct {
	*zap.Logger
}

func NewLog(conf *viper.Viper) *Logger {
	// log address "out.log" User-defined
	lv := conf.GetString("log.log_level")
	var level zapcore.Level
	//debug<info<warn<error<fatal<panic
	switch lv {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	// 配置日志轮转
	baseLogPath := conf.GetString("log.log_file_name")
	if baseLogPath == "" {
		baseLogPath = "logs/app.log"
	}

	// 确保日志目录存在
	logDir := filepath.Dir(baseLogPath)
	if err := ensureDir(logDir); err != nil {
		panic("Failed to create log directory")
	}

	// 移除文件扩展名以便添加日期后缀
	ext := filepath.Ext(baseLogPath)
	base := strings.TrimSuffix(baseLogPath, ext)

	// 设置日志轮转规则
	// 文件名格式: basename-YYYY-MM-DD.log
	// 每天轮转一次或文件大小超过7MB时轮转
	rotationTime := 24 * time.Hour
	maxAge := time.Duration(conf.GetInt("log.max_age")) * 24 * time.Hour
	rotationSize := int64(conf.GetInt("log.max_size")) * 1024 * 1024 // 转换为字节

	writer, err := rotatelogs.New(
		fmt.Sprintf("%s-%%Y-%%m-%%d%s", base, ext),
		rotatelogs.WithMaxAge(maxAge),             // 文件最大保存时间
		rotatelogs.WithRotationTime(rotationTime), // 日志轮转间隔
		rotatelogs.WithRotationSize(rotationSize), // 文件大小轮转
		rotatelogs.WithLocation(time.Local),       // 使用本地时区
	)

	if err != nil {
		panic("Failed to initialize rotatelogs")
	}

	var encoder zapcore.Encoder
	if conf.GetString("log.encoding") == "console" {
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "Logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
			EncodeTime:     timeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		})
	} else {
		encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     timeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(writer)), // Print to console and file
		level,
	)
	if conf.GetString("env") != "prod" {
		return &Logger{zap.New(core, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
	}
	return &Logger{zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	//enc.AppendString(t.Format("2006-01-02 15:04:05"))
	enc.AppendString(t.Format("2006-01-02 15:04:05.000000000"))
}

// ensureDir 确保目录存在
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// WithValue Adds a field to the specified context
func (l *Logger) WithValue(ctx context.Context, fields ...zapcore.Field) context.Context {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
		c.Request = c.Request.WithContext(context.WithValue(ctx, ctxLoggerKey, l.WithContext(ctx).With(fields...)))
		return c
	}
	return context.WithValue(ctx, ctxLoggerKey, l.WithContext(ctx).With(fields...))
}

// WithContext Returns a zap instance from the specified context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	zl := ctx.Value(ctxLoggerKey)
	ctxLogger, ok := zl.(*zap.Logger)
	if ok {
		return &Logger{ctxLogger}
	}
	return l
}
