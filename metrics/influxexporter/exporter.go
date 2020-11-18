package influxexporter

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/influxdata/influxdb/client/v2"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"
)

var log = logging.Logger("influxexporter")

const (
	influxAddrEnvVar     = "INFLUX_ADDR"
	influxUserEnvVar     = "INFLUX_USER"
	influxPassEnvVar     = "INFLUX_PASS"
	influxDatabaseEnvVar = "INFLUX_DATABASE"
)

func NewExporter(commonTags map[string]string) (view.Exporter, func(), error) {
	addr := os.Getenv(influxAddrEnvVar)
	user := os.Getenv(influxUserEnvVar)
	pass := os.Getenv(influxPassEnvVar)
	database := os.Getenv(influxDatabaseEnvVar)

	if addr == "" || user == "" || pass == "" || database == "" {
		return nil, nil, xerrors.New("environment variables not set")
	}

	ip := GetMyExternalIP()
	if len(ip) == 0 {
		return nil, nil, xerrors.New("failed to lookup local IP")
	}

	if commonTags == nil {
		commonTags = make(map[string]string)
	}
	commonTags["ip"] = ip

	influx, err := InfluxClient(addr, user, pass)
	if err != nil {
		return nil, nil, err
	}

	return &exporter{
			influxClient: influx,
			database:     database,
			onError: func(err error) {
				log.Warnf("failed to export view: %s", err)
			},
			commonTags: commonTags,
		}, func() {
			influx.Close()
			log.Info("client closed")
		}, nil
}

type exporter struct {
	influxClient client.Client
	database     string
	onError      func(error)
	commonTags   map[string]string
}

func (e *exporter) ExportView(viewData *view.Data) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  e.database,
		Precision: "s",
	})
	if err != nil {
		e.onError(err)
		return
	}

	for _, row := range viewData.Rows {
		fields := make(map[string]interface{})

		switch d := row.Data.(type) {
		case *view.CountData:
			fields["value"] = float64(d.Value)
		case *view.DistributionData:
			fields["min"] = d.Min
			fields["max"] = d.Max
			fields["mean"] = d.Mean
			fields["count"] = d.Count
		case *view.LastValueData:
			fields["value"] = float64(d.Value)
		case *view.SumData:
			fields["value"] = float64(d.Value)
		default:
			e.onError(fmt.Errorf("unknown AggregationData type: %T", row.Data))
			return
		}

		tagsMap := make(map[string]string)
		appendAndReplace(tagsMap, e.commonTags)
		appendAndReplace(tagsMap, convertTags(row.Tags))

		pt, err := client.NewPoint(viewData.View.Name, tagsMap, fields, viewData.End)
		if err != nil {
			e.onError(err)
		}
		bp.AddPoint(pt)
	}

	err = e.influxClient.Write(bp)
	if err != nil {
		e.onError(err)
	}
}

func appendAndReplace(dst, src map[string]string) {
	if dst == nil {
		return
	}

	for k, v := range src {
		dst[k] = v
	}
}

func convertTags(tags []tag.Tag) map[string]string {
	res := make(map[string]string)
	for _, tag := range tags {
		res[tag.Key.Name()] = tag.Value
	}
	return res
}

func InfluxClient(addr, user, pass string) (client.Client, error) {
	return client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: user,
		Password: pass,
	})
}

var Servers = []string{
	"https://api.ipify.org?format=text",
	"http://myexternalip.com/raw",
	"https://ident.me/",
	"http://bot.whatismyipaddress.com/",
}

func GetMyExternalIP() (ip string) {
	for _, server := range Servers {
		resp, err := http.Get(server)
		if err != nil {
			log.Warnf("failed to lookup my external IP from %s: %v", server, err)
			continue
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warnf("failed to read response from %s: %v", server, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		ip = string(buf)
		if net.ParseIP(ip) != nil {
			break
		}
	}
	return
}
