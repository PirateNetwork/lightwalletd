package common

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	metrics *PrometheusMetrics
	log     *logrus.Entry
)

// Handle http(s) downloads for zcash params
func paramsHandler(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "sapling-output.params") {
		metrics.TotalSaplingParamsCounter.Inc()
		log.WithFields(logrus.Fields{
			"method": "params",
			"param":  "sapling-output",
		}).Info("ParamsHandler")

		http.Redirect(w, req, "https://z.cash/downloads/sapling-output.params", 301)
		return
	}

	if strings.HasSuffix(req.URL.Path, "sapling-spend.params") {
		log.WithFields(logrus.Fields{
			"method": "params",
			"param":  "sapling-spend",
		}).Info("ParamsHandler")

		http.Redirect(w, req, "https://z.cash/downloads/sapling-spend.params", 301)
		return
	}

	if strings.HasSuffix(req.URL.Path, "sprout-groth16.params") {
		log.WithFields(logrus.Fields{
			"method": "params",
			"param":  "sprout",
		}).Info("ParamsHandler")
		metrics.TotalSproutParamsCounter.Inc()

		http.Redirect(w, req, "https://z.cash/downloads/sprout-groth16.params", 301)
		return
	}

	http.Error(w, "Not Found", 404)
}

// ParamsDownloadHandler Listens on port 8090 for download requests for params
func ParamsDownloadHandler(prommetrics *PrometheusMetrics, logger *logrus.Entry, port string) {
	metrics = prommetrics
	log = logger

	http.HandleFunc("/params/", paramsHandler)

	http.ListenAndServe(port, nil)
}
