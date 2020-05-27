package grafana

const (
	LABEL_DASHBOARD_TITLE Label = "grafana_dashboard_title"
	LABEL_DATASOURCE      Label = "grafana_datasource"
	LABEL_TITLE           Label = "grafana_graph_title"
)

type Label string

var GrafanaLabels = []Label{LABEL_DASHBOARD_TITLE, LABEL_DATASOURCE, LABEL_TITLE}
