// RemoteWrite processes copied from https://gist.github.com/kshcherban/918b72d9ad5519aedcdc44c02c246a02
package prometheus

import "bytes"
import "context"
import "fmt"
import "net/http"

import "github.com/golang/snappy"
import "github.com/prometheus/prometheus/prompb"

func RemoteWrite(ctx context.Context, url string, orgId string, timeseries []prompb.TimeSeries) error {
    writeRequest := &prompb.WriteRequest{ Timeseries : timeseries } 
    data, err := writeRequest.Marshal()

    if err != nil {
	    return fmt.Errorf("Marshal error: %w", err)
	}

    compressed := snappy.Encode(nil, data)

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(compressed))

    req.Header.Set("Content-Type", "application/x-protobuf")
    req.Header.Set("Content-Encoding", "snappy")
    req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

    if orgId != "" {
        req.Header.Set("X-Scope-OrgID", orgId)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    if resp.StatusCode/100 != 2 {
        return fmt.Errorf("Status Code: %d", resp.StatusCode)
    }

    return nil
}
