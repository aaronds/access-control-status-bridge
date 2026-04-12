package messages

import mqtt "github.com/eclipse/paho.mqtt.golang"

import "context"
import "time"
import "log"
import "encoding/hex"
import "encoding/binary"
import "encoding/json"
import "errors"
import "strconv"

type EnvPmHandler struct {
    Inbox chan RawMessage
    Outbox chan EnvPm
    Export chan []EnvPm
    context context.Context
}


func (envPmH EnvPmHandler) In () chan RawMessage {
   return envPmH.Inbox
}

func (envPmH EnvPmHandler) Poll () {
    var toExport []EnvPm;
    exportTicker := time.NewTicker(30 * time.Second)

    for {
        select {
            case rawMessage := <- envPmH.Inbox:
                var envPm EnvPm;
                var err error;

                switch rawMessage.Version {
                    case "v1.0.1":
                        envPm, err = DecodeEnvPmV1M0P1(rawMessage) 
                    default:
                        envPm, err = DecodeEnvPmV1M0P1(rawMessage) 
                }

                if err != nil {
                    log.Println(err)
                    break;
                }

                toExport = append(toExport, envPm)
                outputEnvPmMessage(envPmH, envPm)

            case <- exportTicker.C:
                exportEnvPmMessages(envPmH, toExport);
                toExport = toExport[:0]

            case <- envPmH.context.Done():
                return;
        }
    }
}

func (envPmH EnvPmHandler) Push (publishClient mqtt.Client, publishTopicEnvPm string) {
    for {
        select {
            case envPm := <- envPmH.Outbox:
                buff, jsonErr := json.Marshal(envPm)

                if publishClient != nil && publishClient.IsConnectionOpen() {

                    if jsonErr == nil {
                        publishClient.Publish(publishTopicEnvPm + "/by-id/" + envPm.Id, 0, false, buff)
                        publishClient.Publish(publishTopicEnvPm + "/by-location/" + envPm.LocationName + "/all", 0, false, buff)
                    } else {
                        log.Println(jsonErr)
                    }

                    publishClient.Publish(publishTopicEnvPm + "/by-location/" + envPm.LocationName + "/temperature", 0, false, strconv.FormatFloat(envPm.Temperature,'f', 2, 64))

                    if !envPm.Obstructed {
                        publishClient.Publish(publishTopicEnvPm + "/by-location/" + envPm.LocationName + "/pm/2", 0, false, strconv.FormatInt(int64(envPm.Pm2), 10))
                        publishClient.Publish(publishTopicEnvPm + "/by-location/" + envPm.LocationName + "/pm/10", 0, false, strconv.FormatInt(int64(envPm.Pm10), 10))
                    }
                } else {
                    //log.Println("No publish connection")
                }

            case <- envPmH.context.Done():
                return
        }
    }
} 

type EnvPm struct {
    Id string `json:"id"`
    Version string `json:"version"`
    Pm1 uint16 `json:"pm1"`
    Pm2 uint16 `json:"pm2_5"`
    Pm10 uint16 `json:"pm10"`
    Temperature float64 `json:"temperature"`
    RelativeHumidity float64 `json:"relativeHumidity"`
    Pressure float64 `json:"pressure"`
    Obstructed bool `json:"obstructed"`
    Location uint16 `json:"location"`
    LocationName string `json:"locationName"`
    Ts time.Time `json:"ts"`
}

func exportEnvPmMessages(envPmH EnvPmHandler, toExport []EnvPm) {
    select {
        case envPmH.Export <- toExport:
            return;
        default:
            <- envPmH.Export
            envPmH.Export <- toExport;
    }
}

func CreateEnvPmHandler(ctx context.Context) *EnvPmHandler {
    envPmHandler := EnvPmHandler{
        Inbox : make(chan RawMessage, 10),
        Outbox : make(chan EnvPm, 100),
        Export : make(chan []EnvPm, 10),
        context : ctx,
    }

   return &envPmHandler
}

func DecodeEnvPmV1M0P1(raw RawMessage) (EnvPm, error) {

    envPm := EnvPm{ Ts : raw.Received, Version : "1.0.1" , LocationName : "unknown/unknown"}

    if len(raw.Payload) < 22 {
        return envPm, errors.New("Payload incorrect size: " + string(len(raw.Payload)))
    }

    envPm.Id = hex.EncodeToString(raw.Payload[0:6])
    envPm.Pm1 = binary.LittleEndian.Uint16(raw.Payload[6:8])
    envPm.Pm2 = binary.LittleEndian.Uint16(raw.Payload[8:10])
    envPm.Pm10 = binary.LittleEndian.Uint16(raw.Payload[10:12])
    envPm.Temperature = (float64(int16(binary.LittleEndian.Uint16(raw.Payload[12:14])))) / 100
    envPm.RelativeHumidity = (float64(binary.LittleEndian.Uint16(raw.Payload[14:16]))) / 100
    envPm.Pressure = (float64(binary.LittleEndian.Uint16(raw.Payload[16:18]))) / 100
    flags := binary.LittleEndian.Uint16(raw.Payload[18:20])
    envPm.Obstructed = (flags & 1) > 0
    
    envPm.Location = binary.LittleEndian.Uint16(raw.Payload[20:22])

    switch envPm.Location {
        case 110:
            envPm.LocationName = "woodshop/main"
        default:
            envPm.LocationName = "unknown/unknown"
    }

    return envPm, nil
}


func outputEnvPmMessage(envPmH EnvPmHandler, envPm EnvPm) {
    select {
        case envPmH.Outbox <- envPm:
            return;
        default:
            <- envPmH.Outbox
            envPmH.Outbox <- envPm
    }
}

