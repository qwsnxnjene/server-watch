package handlers

import (
	"encoding/json"
	"testing"
)

func BenchmarkJSONEncode(b *testing.B) {
	metrics := MetricsResponse{
		CPUPercent:  35.2,
		MemPercent:  62.1,
		MemUsedMB:   4096,
		MemTotalMB:  8192,
		DiskPercent: 48.7,
		DiskUsedGB:  120,
		DiskTotalGB: 250,
	}

	response := HistoryResponse{Metrics: make([]MetricsResponse, 0, 20)}

	for range 20 {
		response.Metrics = append(response.Metrics, metrics)
	}

	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(response); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	metrics := MetricsResponse{
		CPUPercent:  35.2,
		MemPercent:  62.1,
		MemUsedMB:   4096,
		MemTotalMB:  8192,
		DiskPercent: 48.7,
		DiskUsedGB:  120,
		DiskTotalGB: 250,
	}

	resp := HistoryResponse{Metrics: make([]MetricsResponse, 0, 20)}

	for range 20 {
		resp.Metrics = append(resp.Metrics, metrics)
	}
	data, err := json.Marshal(resp)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		var response HistoryResponse

		if err := json.Unmarshal(data, &response); err != nil {
			b.Fatal(err)
		}
	}
}
