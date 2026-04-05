package messages

import "context"
import "time"
import "log"
import "encoding/hex"
import "encoding/binary"
import "errors"

type AcsErrorHandler struct {
    Inbox chan RawMessage
    //Outbox chan AcsError
    Export chan []AcsError
    context context.Context
}


func (errorH AcsErrorHandler) In () chan RawMessage {
   return errorH.Inbox
}

func (errorH AcsErrorHandler) Poll () {
    var toExport []AcsError;
    exportTicker := time.NewTicker(30 * time.Second)

    for {
        select {
            case rawMessage := <- errorH.Inbox:
                var errorMsg AcsError;
                var err error;

                switch rawMessage.Version {
                    case "v1.0.1":
                        errorMsg, err = DecodeAcsErrorV1M0P1(rawMessage) 
                    default:
                        errorMsg, err = DecodeAcsErrorV1M0P1(rawMessage) 
                }

                if err != nil {
                    log.Println(err)
                    break;
                }

                toExport = append(toExport, errorMsg)

            case <- exportTicker.C:
                exportAcsErrorMessages(errorH, toExport);
                toExport = toExport[:0]

            case <- errorH.context.Done():
                return;
        }
    }
}

type AcsError struct {
    Id string
    Tag uint16
    Error uint16
    Ts time.Time
}

func exportAcsErrorMessages(errorH AcsErrorHandler, toExport []AcsError) {
    select {
        case errorH.Export <- toExport:
            return;
        default:
            <- errorH.Export
            errorH.Export <- toExport;
    }
}

func CreateAcsErrorHandler(ctx context.Context) *AcsErrorHandler {
    errorHandler := AcsErrorHandler{
        Inbox : make(chan RawMessage, 10),
        //Outbox : make(chan AcsError, 100),
        Export : make(chan []AcsError, 10),
        context : ctx,
    }

   return &errorHandler
}

func DecodeAcsErrorV1M0P1(raw RawMessage) (AcsError, error) {

    acsError := AcsError{ Ts : raw.Received }

    if len(raw.Payload) < 10 {
        return acsError, errors.New("ACS Error Payload incorrect size: " + string(len(raw.Payload)))
    }

    acsError.Id = hex.EncodeToString(raw.Payload[0:6])
    acsError.Tag = binary.LittleEndian.Uint16(raw.Payload[6:8])
    acsError.Error = binary.LittleEndian.Uint16(raw.Payload[8:10])

    return acsError, nil
}
