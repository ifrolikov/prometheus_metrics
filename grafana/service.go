package grafana

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"github.com/grafana-tools/sdk"
	"github.com/pkg/errors"
	"net/http"
	"regexp"
)

var dashboardNotFoundRegexp = regexp.MustCompile("Dashboard not found")

type Service struct {
	datasource                             string
	panelWitdh                             int
	panelHeight                            int
	client                                 *sdk.Client
	existingGraphsInDashboardsRuntimeCache map[string][]string
}

func NewService(apiUrl string, authKey string, datasource string) *Service {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	result := &Service{
		client:                                 sdk.NewClient(apiUrl, authKey, httpClient),
		panelWitdh:                             24,
		panelHeight:                            7,
		datasource:                             datasource,
		existingGraphsInDashboardsRuntimeCache: map[string][]string{},
	}

	return result
}

func (this *Service) PushCounterGraph(dashboard string,
	metricName string,
	title string,
	namespace string,
	subsystem string,
	ctx context.Context,
	datasource *string) error {
	fullMetricName := namespace + "_" + subsystem + "_" + metricName;
	metricExpression := `sum(xincrease(` + fullMetricName + `[1h]))`
	if datasource == nil {
		datasource = &this.datasource
	}
	return this.pushGraph(dashboard,
		title,
		fullMetricName,
		metricExpression,
		"short",
		*datasource,
		ctx)
}

func (this *Service) PushCustomCounterGraph(dashboard string,
	fullMetricName string,
	title string,
	ctx context.Context,
	datasource *string) error {
	metricExpression := `sum(xincrease(` + fullMetricName + `[1h]))`
	if datasource == nil {
		datasource = &this.datasource
	}
	return this.pushGraph(dashboard,
		title,
		fullMetricName,
		metricExpression,
		"short",
		*datasource,
		ctx)
}

func (this *Service) PushTimerGraph(dashboard string,
	metricName string,
	title string,
	namespace string,
	subsystem string,
	ctx context.Context,
	datasource *string) error {
	fullMetricName := namespace + "_" + subsystem + "_" + metricName;
	metricExpression := `max by(quantile)(` + fullMetricName + `)`
	if datasource == nil {
		datasource = &this.datasource
	}
	return this.pushGraph(dashboard,
		title,
		fullMetricName,
		metricExpression,
		"ns",
		*datasource,
		ctx)
}

func (this *Service) PushCustomTimerGraph(dashboard string,
	fullMetricName string,
	title string,
	ctx context.Context,
	datasource *string) error {
	metricExpression := `max by(quantile)(` + fullMetricName + `)`
	if datasource == nil {
		datasource = &this.datasource
	}
	return this.pushGraph(dashboard,
		title,
		fullMetricName,
		metricExpression,
		"ns",
		*datasource,
		ctx)
}

func (this *Service) pushGraph(dashboard string,
	title string,
	fullMetricName string,
	metricExpression string,
	yAxisUnit string,
	datasource string,
	ctx context.Context) error {
	if graphs, ok := this.existingGraphsInDashboardsRuntimeCache[dashboard]; ok {
		for _, graph := range graphs {
			if graph == fullMetricName {
				return nil
			}
		}
	}
	board, err := this.initBoard(
		this.initUID(dashboard),
		dashboard,
		ctx)
	if err != nil {
		return err
	}

	for _, panel := range board.Panels {
		for _, target := range panel.GraphPanel.Targets {
			metricReg := regexp.MustCompile(fullMetricName)
			if len(metricReg.FindStringSubmatch(target.Expr)) > 0 {
				// Уже добавлено
				this.addGraphToRuntimeCache(fullMetricName, dashboard)
				return nil
			}
		}
	}
	left, top := this.calculateLeftTopCorner(board)
	panelId := this.calculatePanelId(board)

	graph := sdk.NewGraph(title)
	graph.ID = panelId
	gridPos := struct {
		H *int `json:"h,omitempty"`
		W *int `json:"w,omitempty"`
		X *int `json:"x,omitempty"`
		Y *int `json:"y,omitempty"`
	}{
		H: &this.panelHeight,
		W: &this.panelWitdh,
		X: &left,
		Y: &top,
	}
	graph.GridPos = gridPos
	graph.Yaxes = []sdk.Axis{
		{Format: yAxisUnit, Min: &sdk.FloatString{Value: 0.0, Valid: true}, Show: true, LogBase: 1},
		{Format: "short", Show: true, LogBase: 1},
	}
	graph.Xaxis = sdk.Axis{Show: true, Format: "time", LogBase: 1}
	graph.Fill = 1
	graph.Linewidth = 1
	renderer := "flot"
	graph.Renderer = &renderer
	graph.Datasource = &datasource
	dashLength := uint(10)
	graph.DashLength = &dashLength
	dashes := false
	graph.Dashes = &dashes
	graph.GraphPanel.NullPointMode = "null"
	spaceLength := uint(10)
	graph.SpaceLength = &spaceLength
	graph.Tooltip = sdk.Tooltip{
		Shared:    true,
		Sort:      0,
		ValueType: "individual",
	}
	graph.Bars = true
	graph.Legend = sdk.Legend{
		Show:   true,
		Max:    true,
		Min:    true,
		Avg:    true,
		Values: true,
		//AlignAsTable: true,
	}
	graph.AliasColors = []string{}

	graph.AddTarget(&sdk.Target{
		Interval:     "",
		Expr:         metricExpression,
		RefID:        "A",
		LegendFormat: "for {{quantile}}pp",
	})
	board.Panels = append(board.Panels, graph)

	status, err := this.client.SetDashboard(ctx, *board, sdk.SetDashboardParams{
		Overwrite: false,
	})

	if err != nil {
		return err
	}
	if *status.Status != "success" {
		return errors.New(fmt.Sprintf("status not is success: %s", *status.Status))
	}

	this.addGraphToRuntimeCache(fullMetricName, dashboard)
	return nil
}

func (this *Service) addGraphToRuntimeCache(fullMetricName string, dashboard string) {
	if _, ok := this.existingGraphsInDashboardsRuntimeCache[dashboard]; !ok {
		this.existingGraphsInDashboardsRuntimeCache[dashboard] = []string{}
	}
	this.existingGraphsInDashboardsRuntimeCache[dashboard] = append(this.existingGraphsInDashboardsRuntimeCache[dashboard], fullMetricName)
}

func (this *Service) initBoard(uid string, title string, ctx context.Context) (*sdk.Board, error) {
	board, _, err := this.client.GetDashboardByUID(ctx, uid)
	if err != nil {
		if len(dashboardNotFoundRegexp.FindStringSubmatch(err.Error())) != 0 {
			board = *sdk.NewBoard(title)
			board.Time = sdk.Time{From: "now-24h", To: "now"}
			board.Slug = uid
			board.UID = uid
		} else {
			return nil, err
		}
	}

	board.Annotations = struct {
		List []sdk.Annotation `json:"list"`
	}{List: []sdk.Annotation{}}
	return &board, nil
}

func (this *Service) initUID(title string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(title)))
}

func (this *Service) calculateLeftTopCorner(board *sdk.Board) (int, int) {
	if len(board.Panels) == 0 {
		return 0, 0
	}

	var left, top = 0, 0
	for _, panel := range board.Panels {
		panelTop := *panel.GridPos.H + *panel.GridPos.Y
		if panelTop > top {
			top = panelTop
		}
	}
	return left, top
}

func (this *Service) calculatePanelId(board *sdk.Board) uint {
	id := uint(0)
	for _, panel := range board.Panels {
		if panel.ID > id {
			id = panel.ID
		}
	}
	return id + 1
}
