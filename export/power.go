package export

import "context"
import "log"
//import "fmt"
import "slices"

import "github.com/prometheus/prometheus/prompb"
import "github.com/prometheus/common/model"
import "access-control-status-bridge/messages"
import "access-control-status-bridge/prometheus"

func PowerToPrometheus(ctx context.Context, export chan []messages.Power) {
    for {
        select {
            case powers := <- export:
                timeSeries := powersToTimeSeries("acs", powers)
                err := prometheus.RemoteWrite(ctx, "http://localhost:8090/api/v1/push", "bhs", timeSeries)

                if err == nil {
                    log.Printf("Exported %d power time series", len(timeSeries))
                } else {
                    log.Print(err) 
                }

            case <- ctx.Done():
                return
        }
    }
}

func powersToTimeSeries(site string, powers []messages.Power) []prompb.TimeSeries {
    deviceIds := make([]string,1)
    deviceToPowers := make(map[string][]messages.Power)

    for _, power := range powers {
        if slices.Index(deviceIds, power.Id) < 0 {
            deviceIds = append(deviceIds, power.Id)
        }

        deviceToPowers[power.Id] = append(deviceToPowers[power.Id], power)
    }

    timeSeries := make([]prompb.TimeSeries,0)

    for _, deviceId := range deviceIds {
        sampleCount := len(deviceToPowers[deviceId])

        metricPower := deviceTimeSeries(site, deviceId, "acs_metric_power", sampleCount) 
        metricEnergy := deviceTimeSeries(site, deviceId, "acs_metric_energy", sampleCount) 
        metricIsOn := deviceTimeSeries(site, deviceId, "acs_metric_isOn", sampleCount) 
        metricFrequency := deviceTimeSeries(site, deviceId, "acs_metric_frequency", sampleCount) 
        metricSampleTime := deviceTimeSeries(site, deviceId, "acs_metric_sampleTime", sampleCount) 
        metricZx := deviceTimeSeries(site, deviceId, "acs_metric_zx", sampleCount) 
        metricCurrentMax := deviceTimeSeries(site, deviceId, "acs_metric_currentMax", sampleCount) 

        for i, power := range deviceToPowers[deviceId] {
            isOn := 0.0

            if power.IsOn {
                isOn = 1.0
            }

            metricIsOn.Samples[i] = prompb.Sample{ Value : isOn, Timestamp : int64(model.TimeFromUnix(power.Ts.Unix())) }

            metricPower.Samples[i] = sampleFromUint32(power, power.Power)
            metricEnergy.Samples[i] = sampleFromUint32(power, power.Energy)
            metricFrequency.Samples[i] = prompb.Sample{
                Value : ((float64(power.Zx) / float64(power.Time)) / 1000000) / 2,
                Timestamp : int64(model.TimeFromUnix(power.Ts.Unix())),
            }
            metricSampleTime.Samples[i] = sampleFromUint32(power, power.Time)
            metricZx.Samples[i] = sampleFromUint32(power, power.Zx)
            metricCurrentMax.Samples[i] = sampleFromUint32(power, power.CurrentMax)

        }

        timeSeries = append(timeSeries, metricPower)
        timeSeries = append(timeSeries, metricEnergy)
        timeSeries = append(timeSeries, metricIsOn)
        timeSeries = append(timeSeries, metricFrequency)
        timeSeries = append(timeSeries, metricSampleTime)
        timeSeries = append(timeSeries, metricZx)
        timeSeries = append(timeSeries, metricCurrentMax)
    }

    return timeSeries
}

func sampleFromUint32(power messages.Power, val uint32) prompb.Sample {
    return prompb.Sample{ Value : float64(val), Timestamp : int64(model.TimeFromUnix(power.Ts.Unix())) }
}

func deviceTimeSeries(site string, deviceId string, metric string, sampleCount int) prompb.TimeSeries {
    return prompb.TimeSeries{
        Labels : []prompb.Label{
            {Name : "__name__", Value : metric},
            {Name : "project", Value : "acs"},
            {Name : "site", Value : site},
            {Name : "deviceId", Value : deviceId},
        },
        Samples : make([]prompb.Sample,sampleCount),
    }
}

