package handler

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sharathm18/myapp/internal/middleware"
	"github.com/sharathm18/myapp/internal/server"
	"github.com/sharathm18/myapp/internal/validation"
)

// Handler provides base functionality for all handlers
type Handler struct {
	server *server.Server
}

// NewHandler creates a new base handler
func NewHandler(s *server.Server) Handler {
	return Handler{server: s}
}

// HandlerFunc represents a typed handler function that processes a request and returns a response
type HandlerFunc[Req validation.Validatable, Res any] func(c echo.Context, req Req) (Res, error)

// HandlerFuncNoContent represents a typed handler function that processes a request without returning content
type HandlerFuncNoContent[Req validation.Validatable] func(c echo.Context, req Req) error

// ResponseHandler defines the interface for handling different response types
type ResponseHandler interface {
	Handle(c echo.Context, result interface{}) error
	GetOperation() string
	AddAttributes(span trace.Span, result interface{})
}

// JSONResponseHandler handles JSON responses
type JSONResponseHandler struct {
	status int
}

func (h JSONResponseHandler) Handle(c echo.Context, result interface{}) error {
	return c.JSON(h.status, result)
}

func (h JSONResponseHandler) GetOperation() string {
	return "handler"
}

func (h JSONResponseHandler) AddAttributes(span trace.Span, result interface{}) {
	// http.status_code is already set by tracing middleware
}

// NoContentResponseHandler handles no-content responses
type NoContentResponseHandler struct {
	status int
}

func (h NoContentResponseHandler) Handle(c echo.Context, result interface{}) error {
	return c.NoContent(h.status)
}

func (h NoContentResponseHandler) GetOperation() string {
	return "handler_no_content"
}

func (h NoContentResponseHandler) AddAttributes(span trace.Span, result interface{}) {
	// http.status_code is already set by tracing middleware
}

// FileResponseHandler handles file responses
type FileResponseHandler struct {
	status      int
	filename    string
	contentType string
}

func (h FileResponseHandler) Handle(c echo.Context, result interface{}) error {
	data := result.([]byte)
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+h.filename)
	return c.Blob(h.status, h.contentType, data)
}

func (h FileResponseHandler) GetOperation() string {
	return "handler_file"
}

func (h FileResponseHandler) AddAttributes(span trace.Span, result interface{}) {
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("file.name", h.filename),
			attribute.String("file.content_type", h.contentType),
		)
		if data, ok := result.([]byte); ok {
			span.SetAttributes(attribute.Int("file.size_bytes", len(data)))
		}
	}
}

// handleRequest is the unified handler function that eliminates code duplication
func handleRequest[Req validation.Validatable](
	c echo.Context,
	req Req,
	handler func(c echo.Context, req Req) (interface{}, error),
	responseHandler ResponseHandler,
) error {
	start := time.Now()
	method := c.Request().Method
	path := c.Path()
	route := path

	// Get OpenTelemetry span from context
	span := trace.SpanFromContext(c.Request().Context())
	if span.IsRecording() {
		span.SetAttributes(attribute.String("handler.name", route))
		responseHandler.AddAttributes(span, nil)
	}

	// Get context-enhanced logger
	loggerBuilder := middleware.GetLogger(c).With().
		Str("operation", responseHandler.GetOperation()).
		Str("method", method).
		Str("path", path).
		Str("route", route)

	// Add file-specific fields to logger if it's a file handler
	if fileHandler, ok := responseHandler.(FileResponseHandler); ok {
		loggerBuilder = loggerBuilder.
			Str("filename", fileHandler.filename).
			Str("content_type", fileHandler.contentType)
	}

	logger := loggerBuilder.Logger()

	logger.Info().Msg("handling request")

	// Validation with observability
	validationStart := time.Now()
	if err := validation.BindAndValidate(c, req); err != nil {
		validationDuration := time.Since(validationStart)

		logger.Error().
			Err(err).
			Dur("validation_duration", validationDuration).
			Msg("request validation failed")

		if span.IsRecording() {
			span.RecordError(err)
			span.SetStatus(codes.Error, "validation failed")
			span.SetAttributes(
				attribute.String("validation.status", "failed"),
				attribute.Int64("validation.duration_ms", validationDuration.Milliseconds()),
			)
		}
		return err
	}

	validationDuration := time.Since(validationStart)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("validation.status", "success"),
			attribute.Int64("validation.duration_ms", validationDuration.Milliseconds()),
		)
	}

	logger.Debug().
		Dur("validation_duration", validationDuration).
		Msg("request validation successful")

	// Execute handler with observability
	handlerStart := time.Now()
	result, err := handler(c, req)
	handlerDuration := time.Since(handlerStart)

	if err != nil {
		totalDuration := time.Since(start)

		logger.Error().
			Err(err).
			Dur("handler_duration", handlerDuration).
			Dur("total_duration", totalDuration).
			Msg("handler execution failed")

		if span.IsRecording() {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.String("handler.status", "error"),
				attribute.Int64("handler.duration_ms", handlerDuration.Milliseconds()),
				attribute.Int64("total.duration_ms", totalDuration.Milliseconds()),
			)
		}
		return err
	}

	totalDuration := time.Since(start)

	// Record success metrics and tracing
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("handler.status", "success"),
			attribute.Int64("handler.duration_ms", handlerDuration.Milliseconds()),
			attribute.Int64("total.duration_ms", totalDuration.Milliseconds()),
		)
		responseHandler.AddAttributes(span, result)
	}

	logger.Info().
		Dur("handler_duration", handlerDuration).
		Dur("validation_duration", validationDuration).
		Dur("total_duration", totalDuration).
		Msg("request completed successfully")

	return responseHandler.Handle(c, result)
}

// Handle wraps a handler with validation, error handling, logging, metrics, and tracing
func Handle[Req validation.Validatable, Res any](
	h Handler,
	handler HandlerFunc[Req, Res],
	status int,
	req Req,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		return handleRequest(c, req, func(c echo.Context, req Req) (interface{}, error) {
			return handler(c, req)
		}, JSONResponseHandler{status: status})
	}
}

func HandleFile[Req validation.Validatable](
	h Handler,
	handler HandlerFunc[Req, []byte],
	status int,
	req Req,
	filename string,
	contentType string,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		return handleRequest(c, req, func(c echo.Context, req Req) (interface{}, error) {
			return handler(c, req)
		}, FileResponseHandler{
			status:      status,
			filename:    filename,
			contentType: contentType,
		})
	}
}

// HandleNoContent wraps a handler with validation, error handling, logging, metrics, and tracing for endpoints that don't return content
func HandleNoContent[Req validation.Validatable](
	h Handler,
	handler HandlerFuncNoContent[Req],
	status int,
	req Req,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		return handleRequest(c, req, func(c echo.Context, req Req) (interface{}, error) {
			err := handler(c, req)
			return nil, err
		}, NoContentResponseHandler{status: status})
	}
}
