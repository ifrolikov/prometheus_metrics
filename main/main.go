package main

import (
	"context"
	"github.com/ifrolikov/prometheus_metrics/grafana"
)

func main() {
	grafanaService := grafana.NewService(
		"https://grafana.monitoring.devel.tutu.ru",
		"eyJrIjoibHdERFpQQnAzTmFjbzNyd1c3WDFERFlyQXNINHV1NjgiLCJuIjoiZnJvbGlrb3YiLCJpZCI6MX0=",
	)

	ctx := context.TODO()
	err := grafanaService.PushTimerGraph("Frolikov Dashboard From API",
		"aviaapi_graphql_total_processing_time",
		"Тайминг запросов в монолит для GraphQL",
		"aviaapi",
		"public",
		ctx)
	if err != nil {
		panic(err)
	}

	grafanaService.SetPaasDatasource()
	err = grafanaService.PushCustomCounterGraph("Frolikov Dashboard From API",
		`statsd_avia_api_endpoint_process_count{ednpoint="RefundCalculate",status="error"}`,
		"К-во запросов на расчет возврата в монолит в час",
		ctx)
	if err != nil {
		panic(err)
	}
}
