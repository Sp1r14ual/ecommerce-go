package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer подключается к Jaeger и настраивает глобальный провайдер
func InitTracer(serviceName string, jaegerAddr string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// 1. Создаем экспортер (он будет отправлять данные в Jaeger по gRPC)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(jaegerAddr),
		otlptracegrpc.WithInsecure(), // Локально работаем без SSL
	)
	if err != nil {
		return nil, err
	}

	// 2. Описываем наш сервис (имя, чтобы искать его в Jaeger)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 3. Создаем Tracer Provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// 4. Регистрируем его глобально
	otel.SetTracerProvider(tp)

	// Это ВАЖНО: учим трейсер передавать ID запроса по сети в заголовках
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
