package grafana

type InitData struct {
	ApiUrl string
	AuthKey string
	DefaultDashboard string
	DataSource string
}

type InitMetricData struct {
	Datasource string
}