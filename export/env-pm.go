package export

import "context"
import "log"
//import "fmt"
import "slices"

import "github.com/prometheus/prometheus/prompb"
import "github.com/prometheus/common/model"
import "access-control-status-bridge/messages"
import "access-control-status-bridge/prometheus"

func EnvPmToPrometheus(ctx context.Context, export chan []messages.EnvPm) {
    for {
        select {
            case envPms := <- export:
                timeSeries := envPmsToTimeSeries("acs", envPms)
                err := prometheus.RemoteWrite(ctx, timeSeries)

                if err == nil {
                    log.Printf("Exported %d envPm time series", len(timeSeries))
                } else {
                    log.Print(err) 
                }

            case <- ctx.Done():
                return
        }
    }
}

func envPmsToTimeSeries(site string, envPms []messages.EnvPm) []prompb.TimeSeries {
    deviceIds := make([]string,0)
    deviceToEnvPms := make(map[string][]messages.EnvPm)

    for _, envPm := range envPms {
        if slices.Index(deviceIds, envPm.Id) < 0 {
            deviceIds = append(deviceIds, envPm.Id)
        }

        deviceToEnvPms[envPm.Id] = append(deviceToEnvPms[envPm.Id], envPm)
    }

    timeSeries := make([]prompb.TimeSeries,0)

    for _, deviceId := range deviceIds {
        sampleCount := len(deviceToEnvPms[deviceId])
        firstRecord := deviceToEnvPms[deviceId][0]

        metricEnvPmPm1 := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_pm1", sampleCount)) 
        metricEnvPmPm2 := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_pm2_5", sampleCount)) 
        metricEnvPmPm10 := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_pm10", sampleCount)) 

        metricEnvPmTemperature := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_temperature", sampleCount)) 
        metricEnvPmRelativeHumidity := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_relative_humidity", sampleCount)) 
        metricEnvPmPressure := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_pressure", sampleCount)) 

        metricEnvPmObstructed := addLocation(firstRecord.Location, deviceTimeSeries(site, deviceId, "env_pm_obstructed", sampleCount)) 

        for i, envPm := range deviceToEnvPms[deviceId] {
            isObstructed := 0.0

            if envPm.Obstructed {
                isObstructed = 1.0
            }

            metricEnvPmObstructed.Samples[i] = prompb.Sample{ Value : isObstructed, Timestamp : int64(model.TimeFromUnix(envPm.Ts.Unix())) }

            metricEnvPmPm1.Samples[i] = sampleFromUint16(envPm.Ts, envPm.Pm1)
            metricEnvPmPm2.Samples[i] = sampleFromUint16(envPm.Ts, envPm.Pm2)
            metricEnvPmPm10.Samples[i] = sampleFromUint16(envPm.Ts, envPm.Pm10)

            metricEnvPmTemperature.Samples[i] = sampleFromFloat(envPm.Ts, envPm.Temperature)
            metricEnvPmRelativeHumidity.Samples[i] = sampleFromFloat(envPm.Ts, envPm.RelativeHumidity)
            metricEnvPmPressure.Samples[i] = sampleFromFloat(envPm.Ts, envPm.Pressure)
        }

        timeSeries = append(timeSeries, metricEnvPmPm1)
        timeSeries = append(timeSeries, metricEnvPmPm2)
        timeSeries = append(timeSeries, metricEnvPmPm10)
        timeSeries = append(timeSeries, metricEnvPmTemperature)
        timeSeries = append(timeSeries, metricEnvPmRelativeHumidity)
        timeSeries = append(timeSeries, metricEnvPmPressure)
        timeSeries = append(timeSeries, metricEnvPmObstructed)
    }

    return timeSeries
}
