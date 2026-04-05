package export

import "context"
import "log"
//import "fmt"
import "slices"

import "github.com/prometheus/prometheus/prompb"
import "github.com/prometheus/common/model"
import "access-control-status-bridge/messages"
import "access-control-status-bridge/prometheus"

func ModeToPrometheus(ctx context.Context, export chan []messages.Mode) {
    for {
        select {
            case modes := <- export:
                timeSeries := modesToTimeSeries("acs", modes)
                err := prometheus.RemoteWrite(ctx, timeSeries)

                if err == nil {
                    log.Printf("Exported %d mode time series", len(timeSeries))
                } else {
                    log.Print(err) 
                }

            case <- ctx.Done():
                return
        }
    }
}

func modesToTimeSeries(site string, modes []messages.Mode) []prompb.TimeSeries {
    deviceIds := make([]string,0)
    deviceToModes := make(map[string][]messages.Mode)

    for _, mode := range modes {
        if slices.Index(deviceIds, mode.Id) < 0 {
            deviceIds = append(deviceIds, mode.Id)
        }

        deviceToModes[mode.Id] = append(deviceToModes[mode.Id], mode)
    }

    timeSeries := make([]prompb.TimeSeries,0)

    for _, deviceId := range deviceIds {
        sampleCount := len(deviceToModes[deviceId])

        metricUnlocked := prompb.TimeSeries{
            Labels : []prompb.Label{
                {Name : "__name__", Value : "acs_metric_unlocked"},
                {Name : "project", Value : "acs"},
                {Name : "site", Value : site},
                {Name : "deviceId", Value : deviceId },
            },
            Samples : make([]prompb.Sample, sampleCount),
        }

        metricInUse := prompb.TimeSeries{
            Labels : []prompb.Label{
                {Name : "__name__", Value : "acs_metric_inUse"},
                {Name : "project", Value : "acs"},
                {Name : "site", Value : site},
                {Name : "deviceId", Value : deviceId },
            },
            Samples : make([]prompb.Sample,sampleCount),
        }

        energyTotal := prompb.TimeSeries{
            Labels : []prompb.Label{
                {Name : "__name__", Value : "acs_metric_energyTotal"},
                {Name : "project", Value : "acs"},
                {Name : "site", Value : site},
                {Name : "deviceId", Value : deviceId },
            },
            Samples : make([]prompb.Sample,sampleCount),
        }

        for i, mode := range deviceToModes[deviceId] {
            unlocked := 0.0
            inUse := 0.0

            if mode.Mode == "CONTROLLER_MODE_UNLOCKED" || mode.Mode == "CONTROLLER_MODE_IN_USE" {
                unlocked = 1.0
            }

            metricUnlocked.Samples[i] = prompb.Sample{ Value : unlocked, Timestamp : int64(model.TimeFromUnix(mode.Ts.Unix())) }

            if mode.Mode == "CONTROLLER_MODE_IN_USE" {
                inUse = 1.0
            }

            metricInUse.Samples[i] = prompb.Sample{ Value : inUse, Timestamp : int64(model.TimeFromUnix(mode.Ts.Unix())) }

            energyTotal.Samples[i] = prompb.Sample{ Value : float64(mode.EnergyTotal), Timestamp : int64(model.TimeFromUnix(mode.Ts.Unix())) }

        }

        timeSeries = append(timeSeries, metricUnlocked)
        timeSeries = append(timeSeries, metricInUse)
    }

    return timeSeries
}
