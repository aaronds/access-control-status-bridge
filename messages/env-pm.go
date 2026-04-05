package messages

import "context"
import "time"
import "log"
import "encoding/hex"
import "encoding/binary"
import "errors"

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

            case <- exportTicker.C:
                exportEnvPmMessages(envPmH, toExport);
                toExport = toExport[:0]

            case <- envPmH.context.Done():
                return;
        }
    }
}

type EnvPm struct {
    Id string
    Pm1 uint16
    Pm2 uint16
    Pm10 uint16
    Temperature float64
    RelativeHumidity float64
    Pressure float64
    Obstructed bool
    Location uint16
    Ts time.Time
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

    envPm := EnvPm{ Ts : raw.Received }

    if len(raw.Payload) < 20 {
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

    return envPm, nil
}
