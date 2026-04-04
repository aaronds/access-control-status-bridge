package messages

import "context"
import "time"
import "log"
import "encoding/hex"
import "encoding/binary"
import "errors"

type PowerHandler struct {
    Inbox chan RawMessage
    //Outbox chan Power
    Export chan []Power
    context context.Context
}


func (powerH PowerHandler) In () chan RawMessage {
   return powerH.Inbox
}

func (powerH PowerHandler) Poll () {
    var toExport []Power;
    exportTicker := time.NewTicker(30 * time.Second)

    for {
        select {
            case rawMessage := <- powerH.Inbox:
                var power Power;
                var err error;

                switch rawMessage.Version {
                    case "v1.0.1":
                        power, err = DecodePowerV1M0P1(rawMessage) 
                    default:
                        power, err = DecodePowerV1M0P1(rawMessage) 
                }

                if err != nil {
                    log.Println(err)
                    break;
                }

                toExport = append(toExport, power)

            case <- exportTicker.C:
                exportPowerMessages(powerH, toExport);
                toExport = toExport[:0]

            case <- powerH.context.Done():
                return;
        }
    }
}


type Power struct {
    Id string
    Energy uint32
    Power uint32
    Time uint32
    CurrentMax uint32
    Zx uint32
    Voltage uint16
    VoltageType uint8
    IsOn bool
    Ts time.Time
}


func exportPowerMessages(powerH PowerHandler, toExport []Power) {
    select {
        case powerH.Export <- toExport:
            return;
        default:
            <- powerH.Export
            powerH.Export <- toExport;
    }
}

func CreatePowerHandler(ctx context.Context) *PowerHandler {
    powerHandler := PowerHandler{
        Inbox : make(chan RawMessage, 10),
        //Outbox : make(chan Power, 100),
        Export : make(chan []Power, 10),
        context : ctx,
    }

   return &powerHandler
}

func DecodePowerV1M0P1(raw RawMessage) (Power, error) {

    power := Power{ Ts : raw.Received }

    if len(raw.Payload) < 29 {
        return power, errors.New("Payload incorrect size: " + string(len(raw.Payload)))
    }

    power.Id = hex.EncodeToString(raw.Payload[0:6])
    power.Energy = binary.LittleEndian.Uint32(raw.Payload[6:10])
    power.Power = binary.LittleEndian.Uint32(raw.Payload[10:14])
    power.Time = binary.LittleEndian.Uint32(raw.Payload[14:18])
    power.CurrentMax = binary.LittleEndian.Uint32(raw.Payload[18:22])
    power.Zx = binary.LittleEndian.Uint32(raw.Payload[22:26])
    power.Voltage = binary.LittleEndian.Uint16(raw.Payload[26:28])
    power.VoltageType = raw.Payload[28]
    power.IsOn = raw.Payload[29] > 0

    return power, nil
}
