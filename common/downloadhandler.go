package common

import (
	"net/http"
	"strings"
)

var (
	metrics *PrometheusMetrics
)

// Handle http(s) downloads for zcash params
func paramsHandler(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "sapling-output.params") {
		metrics.TotalSaplingParamsCounter.Inc()
		http.Redirect(w, req, "https://z.cash/downloads/sapling-output.params", 301)
		return
	}

	if strings.HasSuffix(req.URL.Path, "sapling-spend.params") {
		http.Redirect(w, req, "https://z.cash/downloads/sapling-spend.params", 301)
		return
	}

	if strings.HasSuffix(req.URL.Path, "sprout-groth16.params") {
		metrics.TotalSproutParamsCounter.Inc()
		http.Redirect(w, req, "https://z.cash/downloads/sprout-groth16.params", 301)
		return
	}

	http.Error(w, "Not Found", 404)
}

// ParamsDownloadHandler Listens on port 8090 for download requests for params
func ParamsDownloadHandler(prommetrics *PrometheusMetrics) {
	metrics = prommetrics
	http.HandleFunc("/params/", paramsHandler)

	http.ListenAndServe(":8090", nil)
}
