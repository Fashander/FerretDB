// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"

	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/handler/middleware"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
)

// MsgRefreshSessions implements `refreshSessions` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgRefreshSessions(connCtx context.Context, req *middleware.MsgRequest) (*middleware.MsgResponse, error) {
	spec, err := req.RawDocument()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	doc, err := spec.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	_, _, err = h.s.CreateOrUpdateByLSID(connCtx, spec)
	if err != nil {
		return nil, err
	}

	ids, err := getSessionIDsParam(doc, doc.Command())
	if err != nil {
		return nil, err
	}

	h.s.CreateOrUpdateSessions(connCtx, ids)

	return middleware.Response(wirebson.MustDocument(
		"ok", float64(1),
	))
}
