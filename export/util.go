package export

import "time"
import "strconv"
import "github.com/prometheus/prometheus/prompb"
import "github.com/prometheus/common/model"

func sampleFromFloat(ts time.Time, val float64) prompb.Sample {
    return prompb.Sample{ Value : val, Timestamp : int64(model.TimeFromUnix(ts.Unix())) }
}

func sampleFromUint16(ts time.Time, val uint16) prompb.Sample {
    return prompb.Sample{ Value : float64(val), Timestamp : int64(model.TimeFromUnix(ts.Unix())) }
}

func sampleFromUint32(ts time.Time, val uint32) prompb.Sample {
    return prompb.Sample{ Value : float64(val), Timestamp : int64(model.TimeFromUnix(ts.Unix())) }
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

func addLocation(location uint16, ts prompb.TimeSeries) prompb.TimeSeries {
    ts.Labels = append(ts.Labels, prompb.Label{ Name : "location", Value : strconv.Itoa(int(location)) }) 
    return ts
}
