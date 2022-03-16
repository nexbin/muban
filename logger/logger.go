package logger

import (
	"BlueBell2/settings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

func Init(conf *settings.LogConfig) (err error) {
	// 从配置中读取日志级别，并且反序列化
	infoLevel := new(zapcore.Level)
	err = infoLevel.UnmarshalText([]byte(conf.Level))
	if err != nil {
		return err
	}
	core := zapcore.NewCore(
		getEncoder(),
		getWriteSyncer(
			conf.FileName,
			conf.MaxSize,
			conf.MaxAge,
			conf.MaxBackups,
		),
		zapcore.DebugLevel,
	)

	l := zap.New(core, zap.AddCaller()) // 创建一个日志记录器
	zap.ReplaceGlobals(l)               // 自定义的替换全局日志
	return
}

// 创建自定义编码器-如何写入日志
func getEncoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder // 更改时间编码
	config.TimeKey = "ts"
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncodeDuration = zapcore.SecondsDurationEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewJSONEncoder(config)
}

// 指定日志写到那里去
//func getWriteSyncer() zapcore.WriteSyncer {
//	file, _ := os.OpenFile("./web_app.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0744)
//	return zapcore.AddSync(file)
//}

func getWriteSyncer(fileName string, maxSize int, maxAge int, maxBackup int) zapcore.WriteSyncer {
	// 支持日志切割
	logger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    maxSize,   // M
		MaxAge:     maxAge,    // 最大备份天数
		MaxBackups: maxBackup, // 备份数量
		LocalTime:  true,
		Compress:   false, // 是否压缩
	}
	return zapcore.AddSync(logger)
}

// GinLogger 接收gin框架默认的日志
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		zap.L().Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}

// GinRecovery recover掉项目可能出现的panic，并使用zap记录相关日志
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					zap.L().Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
