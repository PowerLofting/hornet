package participation

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/serializer"
)

const (
	ParticipationsMinParticipationCount = 1
	ParticipationsMaxParticipationCount = 255
)

var (
	participationsArrayRules = &serializer.ArrayRules{
		Min:            ParticipationsMinParticipationCount,
		Max:            ParticipationsMaxParticipationCount,
		ValidationMode: serializer.ArrayValidationModeNone,
	}

	ErrMultipleEventParticipation = errors.New("multiple participations for the same event")
)

// Participations holds the participation for multiple events.
type Participations struct {
	// Participations holds the participation for multiple events.
	Participations serializer.Serializables
}

func (p *Participations) Deserialize(data []byte, deSeriMode serializer.DeSerializationMode) (int, error) {
	return serializer.NewDeserializer(data).
		ReadSliceOfObjects(func(seri serializer.Serializables) { p.Participations = seri }, deSeriMode, serializer.SeriLengthPrefixTypeAsByte, serializer.TypeDenotationNone, func(_ uint32) (serializer.Serializable, error) {
			// there is no real selector, so we always return a fresh Participation
			return &Participation{}, nil
		}, participationsArrayRules, func(err error) error {
			return fmt.Errorf("unable to deserialize participations: %w", err)
		}).
		AbortIf(func(err error) error {
			if deSeriMode.HasMode(serializer.DeSeriModePerformValidation) {
				seenEvents := make(map[EventID]struct{})
				for _, s := range p.Participations {
					switch participation := s.(type) {
					case *Participation:
						if _, found := seenEvents[participation.EventID]; found {
							return ErrMultipleEventParticipation
						}
						seenEvents[participation.EventID] = struct{}{}
					default:
						return errors.New("invalid participation type")
					}
				}
			}
			return nil
		}).
		Done()
}

func (p *Participations) Serialize(deSeriMode serializer.DeSerializationMode) ([]byte, error) {
	return serializer.NewSerializer().
		AbortIf(func(err error) error {
			if deSeriMode.HasMode(serializer.DeSeriModePerformValidation) {
				if err := participationsArrayRules.CheckBounds(uint(len(p.Participations))); err != nil {
					return fmt.Errorf("unable to serialize participations: %w", err)
				}
				seenEvents := make(map[EventID]struct{})
				for _, s := range p.Participations {
					switch participation := s.(type) {
					case *Participation:
						if _, found := seenEvents[participation.EventID]; found {
							return ErrMultipleEventParticipation
						}
						seenEvents[participation.EventID] = struct{}{}
					default:
						return errors.New("invalid participation type")
					}
				}
			}
			return nil
		}).
		WriteSliceOfObjects(p.Participations, deSeriMode, serializer.SeriLengthPrefixTypeAsByte, nil, func(err error) error {
			return fmt.Errorf("unable to serialize participations: %w", err)
		}).
		Serialize()
}

func (p *Participations) MarshalJSON() ([]byte, error) {
	j := &jsonParticipations{}

	j.Participations = make([]*json.RawMessage, len(p.Participations))
	for i, participation := range p.Participations {
		jsonParticipation, err := participation.MarshalJSON()
		if err != nil {
			return nil, err
		}
		rawJSONParticipation := json.RawMessage(jsonParticipation)
		j.Participations[i] = &rawJSONParticipation
	}

	return json.Marshal(j)
}

func (p *Participations) UnmarshalJSON(bytes []byte) error {
	j := &jsonParticipations{}
	if err := json.Unmarshal(bytes, j); err != nil {
		return err
	}
	seri, err := j.ToSerializable()
	if err != nil {
		return err
	}
	*p = *seri.(*Participations)
	return nil
}

// jsonParticipations defines the JSON representation of Participations.
type jsonParticipations struct {
	// Participations holds the participation for multiple events.
	Participations []*json.RawMessage `json:"participations"`
}

func (j *jsonParticipations) ToSerializable() (serializer.Serializable, error) {
	payload := &Participations{}

	participations := make(serializer.Serializables, len(j.Participations))
	for i, ele := range j.Participations {
		participation := &Participation{}

		rawJSON, err := ele.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("pos %d: %w", i, err)
		}

		if err := json.Unmarshal(rawJSON, participation); err != nil {
			return nil, fmt.Errorf("pos %d: %w", i, err)
		}

		participations[i] = participation
	}
	payload.Participations = participations

	return payload, nil
}
