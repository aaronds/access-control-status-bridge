package export

import "context"
import "log"
import "fmt"
import "slices"
import "strings"
import "strconv"

import "github.com/prometheus/prometheus/prompb"
import "github.com/prometheus/common/model"
import "access-control-status-bridge/messages"
import "access-control-status-bridge/prometheus"

func AcsErrorToPrometheus(ctx context.Context, export chan []messages.AcsError) {
    for {
        select {
            case acsErrors := <- export:
                timeSeries := acsErrorsToTimeSeries("acs", acsErrors)
                err := prometheus.RemoteWrite(ctx, "http://localhost:8090/api/v1/push", "bhs", timeSeries)

                if err == nil {
                    log.Printf("Exported %d acsError time series", len(timeSeries))
                } else {
                    log.Print(err) 
                }

            case <- ctx.Done():
                return
        }
    }
}

func acsErrorsToTimeSeries(site string, acsErrors []messages.AcsError) []prompb.TimeSeries {
    deviceKeys := make([]string,0)
    deviceKeyToAcsErrors := make(map[string][]messages.AcsError)

    for _, acsError := range acsErrors {
        tagStr := strconv.Itoa(int(acsError.Tag))
        errorStr := strconv.Itoa(int(acsError.Error))
        deviceKey := strings.Join([]string{acsError.Id, tagStr, errorStr},"_")

        fmt.Println("deviceKey: " + deviceKey)

        if slices.Index(deviceKeys, deviceKey) < 0 {
            deviceKeys = append(deviceKeys, deviceKey)
        }

        deviceKeyToAcsErrors[deviceKey] = append(deviceKeyToAcsErrors[deviceKey], acsError)
    }

    timeSeries := make([]prompb.TimeSeries,0)

    for _, deviceKey := range deviceKeys {
        sampleCount := len(deviceKeyToAcsErrors[deviceKey])
        firstRecord := deviceKeyToAcsErrors[deviceKey][0]

        metricAcsError := deviceTimeSeries(site, firstRecord.Id, "acs_metric_error", sampleCount) 
        metricAcsError.Labels = append(
            metricAcsError.Labels,
            prompb.Label{ Name : "tag", Value : strconv.Itoa(int(firstRecord.Tag)) },
            prompb.Label{ Name : "error", Value : strconv.Itoa(int(firstRecord.Error)) },
        )

        for i, acsError := range deviceKeyToAcsErrors[deviceKey] {
            metricAcsError.Samples[i] = prompb.Sample{ Value : 1.0, Timestamp : int64(model.TimeFromUnix(acsError.Ts.Unix())) }
        }

        timeSeries = append(timeSeries, metricAcsError)
    }

    return timeSeries
}
