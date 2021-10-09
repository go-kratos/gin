package kgin

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	thttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

const (
	baseContentType = "application"
)

type errorRender struct {
	body        []byte
	contentType string
}

// Render (JSON) writes data with custom ContentType.
func (er *errorRender) Render(w http.ResponseWriter) error {
	_, err := w.Write(er.body)
	return err
}

// WriteContentType (JSON) writes JSON ContentType.
func (er *errorRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", er.contentType)

}

// Error encodes the object to the HTTP response.
func Error(ctx *gin.Context, err error) {
	if err == nil {
		ctx.Status(http.StatusOK)
		return
	}
	ctx.AbortWithStatusJSON(500, errors.FromError(err))
}

func Success(ctx *gin.Context, data interface{}) {
	if data == nil {
		data = struct{}{}
	}

	if msg, ok := data.(proto.Message); ok {
		var m = jsonpb.Marshaler{
			EmitDefaults: true,
		}
		ctx.Writer.WriteHeader(http.StatusOK)
		ctx.Writer.Header().Add("Content-Type", "application/json; charset=utf-8")
		err := m.Marshal(ctx.Writer, msg)
		if err != nil {
			Error(ctx, err)
			return
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

// ContentType returns the content-type with base prefix.
func ContentType(subtype string) string {
	return strings.Join([]string{baseContentType, subtype}, "/")
}

// Middlewares return middlewares wrapper
func Middlewares(m ...middleware.Middleware) gin.HandlerFunc {
	chain := middleware.Chain(m...)
	return func(c *gin.Context) {
		next := func(ctx context.Context, req interface{}) (interface{}, error) {
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			var err error
			if c.Writer.Status() >= 400 {
				err = errors.Errorf(c.Writer.Status(), errors.UnknownReason, errors.UnknownReason)
			}
			return c.Writer, err
		}
		next = chain(next)
		ctx := NewGinContext(c.Request.Context(), c)
		if ginCtx, ok := FromGinContext(ctx); ok {
			thttp.SetOperation(ctx, ginCtx.FullPath())
		}
		next(ctx, c.Request)
	}
}

type ginKey struct{}

// NewGinContext returns a new Context that carries gin.Context value.
func NewGinContext(ctx context.Context, c *gin.Context) context.Context {
	return context.WithValue(ctx, ginKey{}, c)
}

// FromGinContext returns the gin.Context value stored in ctx, if any.
func FromGinContext(ctx context.Context) (c *gin.Context, ok bool) {
	c, ok = ctx.Value(ginKey{}).(*gin.Context)
	return
}

type Validator interface {
	Validate() error
}

func Validate(in interface{}) error {
	if val, ok := in.(Validator); ok {
		return val.Validate()
	}
	return nil
}
