package messages

import "context"
import "time"
import "encoding/hex"
//import "fmt"
import "encoding/binary"
import "errors"

type ModeHandler struct {
    Inbox chan RawMessage
    Outbox chan Mode
    Export chan []Mode
    context context.Context
}


func (modeH ModeHandler) In () chan RawMessage {
   return modeH.Inbox
}

func modeToString(mode byte) string {
    switch (mode) {
        case 0:
            return "CONTROLLER_MODE_INITIALISING"
        case 1:
            return "CONTROLLER_MODE_LOCKED"
        case 2:
            return "CONTROLLER_MODE_UNLOCKED"
        case 3:
            return "CONTROLLER_MODE_IN_USE"
        case 4:
            return "CONTROLLER_MODE_AWAIT_INDUCTOR"
        case 5:
            return "CONTROLLER_MODE_ENROLL"
        default:
            return "CONTROLLER_MODE_UNKNOWN"
    }
}

type Mode struct {
   Id string `json:"id"`
   Mode string `json:"mode"`
   IsOn bool `json:"isOn"`
   IsUsed bool `json:"isUsed"`
   MonitorEnabled bool `json:"monitorEnabled"`
   NfcEnabled bool `json:"nfcEnabled"`
   IsObserver bool `json:"isObserver"`
   ModeChange bool `json:"modeChange"`
   TimeRemaining uint32 `json:"timeRemaining"`
   UnlockedTimeout uint32 `json:"unlockedTimeout"`
   EnergyTotal uint32 `json:"energyTotal"`
   Ts time.Time `json:"ts"`
}


func (modeH ModeHandler) Poll () {
    var toExport []Mode;
    exportTicker := time.NewTicker(30 * time.Second)

    for {
        select {
            case rawMessage := <- modeH.Inbox:
                var mode Mode
                var err error;

                switch rawMessage.Version {
                    case "v1.0.1":
                        mode, err = DecodeModeV1M0P1(rawMessage) 
                    default:
                        mode, err = DecodeModeV1M0P1(rawMessage) 
                }

                if err != nil {
                    break;
                }

                toExport = append(toExport, mode)
                outputModeMessage(modeH, mode)
            case <- exportTicker.C:
                exportModeMessages(modeH, toExport);
                toExport = toExport[:0]

            case <- modeH.context.Done():
                return;
        }
    }
}

func exportModeMessages(modeH ModeHandler, toExport []Mode) {
    select {
        case modeH.Export <- toExport:
            return;
        default:
            <- modeH.Export
            modeH.Export <- toExport;
    }
}

func outputModeMessage(modeH ModeHandler, mode Mode) {
    select {
        case modeH.Outbox <- mode:
            return;
        default:
            <- modeH.Outbox
            modeH.Outbox <- mode
    }
}

func CreateModeHandler(ctx context.Context) *ModeHandler {
    modeHandler := ModeHandler{
        Inbox : make(chan RawMessage, 10),
        Outbox : make(chan Mode, 100),
        Export : make(chan []Mode, 10),
        context : ctx,
    }

   return &modeHandler
}

func DecodeModeV1M0P1(raw RawMessage) (Mode, error) {


    mode := Mode{ Ts : raw.Received }

    if len(raw.Payload) < 22 {
        return mode, errors.New("Payload incorrect size: " + string(len(raw.Payload)))
    }

    mode.Id = hex.EncodeToString(raw.Payload[0:6])
    mode.TimeRemaining = binary.LittleEndian.Uint32(raw.Payload[6:10])
    mode.UnlockedTimeout = binary.LittleEndian.Uint32(raw.Payload[10:14])
    mode.Mode = modeToString(raw.Payload[14])

    flags := raw.Payload[18]

    mode.IsOn = (flags & 1) > 0
    mode.IsUsed = (flags & 2) > 0
    mode.MonitorEnabled = (flags & 4) > 0
    mode.NfcEnabled = (flags & 8) > 0
    mode.IsObserver = (flags & 16) > 0
    mode.ModeChange = (flags & 32) > 0

    if len(raw.Payload) > 22 {
        mode.EnergyTotal = binary.LittleEndian.Uint32(raw.Payload[22:26])
    }

    return mode, nil
}
