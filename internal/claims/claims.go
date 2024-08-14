/*
Copyright Avast Software. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package claims

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/trustbloc/vcs/pkg/dataprotect"
	"github.com/trustbloc/vcs/pkg/restapi/resterr"
	"github.com/trustbloc/vcs/pkg/service/issuecredential"
)

type dataProtector interface {
	Encrypt(ctx context.Context, msg []byte) (*dataprotect.EncryptedData, error)
	Decrypt(ctx context.Context, encryptedData *dataprotect.EncryptedData) ([]byte, error)
}

func EncryptClaims(
	ctx context.Context,
	data map[string]interface{},
	protector dataProtector,
) (*issuecredential.ClaimData, error) {
	bytesData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	encrypted, err := protector.Encrypt(ctx, bytesData)
	if err != nil {
		return nil, resterr.NewSystemError(resterr.DataProtectorComponent, "Encrypt", err)
	}

	return &issuecredential.ClaimData{
		EncryptedData: encrypted,
	}, nil
}

func DecryptClaims(
	ctx context.Context,
	data *issuecredential.ClaimData,
	protector dataProtector,
) (map[string]interface{}, error) {
	resp, err := protector.Decrypt(ctx, data.EncryptedData)
	if err != nil {
		return nil, resterr.NewSystemError(resterr.DataProtectorComponent, "Decrypt", err)
	}

	finalMap := map[string]interface{}{}
	if err = json.Unmarshal(resp, &finalMap); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return finalMap, nil
}