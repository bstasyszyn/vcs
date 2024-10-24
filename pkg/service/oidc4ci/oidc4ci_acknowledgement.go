/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package oidc4ci

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/trustbloc/vcs/pkg/event/spi"
)

type AckService struct {
	cfg *AckServiceConfig
}

var ErrAckExpired = errors.New("expired_ack_id")

type AckServiceConfig struct {
	AckStore   ackStore
	EventSvc   eventService
	EventTopic string
	ProfileSvc profileService
}

func NewAckService(
	cfg *AckServiceConfig,
) *AckService {
	return &AckService{
		cfg: cfg,
	}
}

// CreateAck creates an acknowledgement.
func (s *AckService) CreateAck(
	ctx context.Context,
	ack *Ack,
) (*string, error) {
	if s.cfg.AckStore == nil {
		return nil, nil //nolint:nilnil
	}

	id, err := s.cfg.AckStore.Create(ctx, ack)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

func (s *AckService) HandleAckNotFound(
	ctx context.Context,
	req AckRemote,
) error {
	if req.IssuerIdentifier == "" {
		return errors.New("issuer identifier is empty and ack not found")
	}

	parts := strings.Split(req.IssuerIdentifier, "/")
	if len(parts) < issuerIdentifierParts {
		return errors.New("invalid issuer identifier. expected format https://xxx/{profileID}/{profileVersion}")
	}

	profileID := parts[len(parts)-2]
	profileVersion := parts[len(parts)-1]

	profile, err := s.cfg.ProfileSvc.GetProfile(profileID, profileVersion)
	if err != nil {
		return err
	}

	eventPayload := &EventPayload{
		WebHook:        profile.WebHook,
		ProfileID:      profile.ID,
		ProfileVersion: profile.Version,
		OrgID:          profile.OrganizationID,
	}

	if req.ErrorText != "" {
		eventPayload.ErrorComponent = "wallet"
		eventPayload.Error = req.ErrorText
	}

	err = s.sendEvent(ctx, spi.IssuerOIDCInteractionAckExpired, TxID(req.ID), eventPayload)
	if err != nil {
		return err
	}

	return ErrAckExpired
}

// Ack acknowledges the interaction.
func (s *AckService) Ack(
	ctx context.Context,
	req AckRemote,
) error {
	if s.cfg.AckStore == nil {
		return nil
	}

	ack, err := s.cfg.AckStore.Get(ctx, req.ID)
	if err != nil {
		if errors.Is(err, ErrDataNotFound) {
			return s.HandleAckNotFound(ctx, req)
		}
		return err
	}

	if ack.HashedToken != req.HashedToken {
		return errors.New("invalid token")
	}

	eventPayload := &EventPayload{
		WebHook:        ack.WebHookURL,
		ProfileID:      ack.ProfileID,
		ProfileVersion: ack.ProfileVersion,
		OrgID:          ack.OrgID,
	}

	if req.ErrorText != "" {
		eventPayload.ErrorComponent = "wallet"
		eventPayload.Error = req.ErrorText
	}

	targetEvent, err := s.AckEventMap(req.Status)
	if err != nil {
		return err
	}

	err = s.sendEvent(ctx, targetEvent, ack.TxID, eventPayload)
	if err != nil {
		return err
	}

	if err = s.cfg.AckStore.Delete(ctx, req.ID); err != nil { // not critical
		logger.Errorc(ctx, fmt.Sprintf("failed to delete ack with id[%s]: %s", req.ID, err.Error()))
	}

	return nil
}

func (s *AckService) sendEvent(
	ctx context.Context,
	eventType spi.EventType,
	transactionID TxID,
	ep *EventPayload,
) error {
	event, err := createEvent(eventType, transactionID, ep)
	if err != nil {
		return err
	}

	return s.cfg.EventSvc.Publish(ctx, s.cfg.EventTopic, event)
}

func (s *AckService) AckEventMap(status string) (spi.EventType, error) {
	switch strings.ToLower(status) {
	case "success":
		return spi.IssuerOIDCInteractionAckSucceeded, nil
	case "failure":
		return spi.IssuerOIDCInteractionAckFailed, nil
	case "rejected":
		return spi.IssuerOIDCInteractionAckRejected, nil
	}

	return spi.IssuerOIDCInteractionAckFailed, fmt.Errorf("invalid status: %s", status)
}
