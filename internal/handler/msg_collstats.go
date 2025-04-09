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
	"fmt"

	"github.com/FerretDB/wire/wirebson"

	"github.com/FerretDB/FerretDB/v2/internal/documentdb/documentdb_api"
	"github.com/FerretDB/FerretDB/v2/internal/handler/middleware"
	"github.com/FerretDB/FerretDB/v2/internal/mongoerrors"
	"github.com/FerretDB/FerretDB/v2/internal/util/lazyerrors"
)

// MsgCollStats implements `collStats` command.
//
// The passed context is canceled when the client connection is closed.
func (h *Handler) MsgCollStats(connCtx context.Context, req *middleware.Request) (*middleware.Response, error) {
	spec, err := req.OpMsg.RawDocument()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/78
	doc, err := spec.Decode()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	if _, _, err = h.s.CreateOrUpdateByLSID(connCtx, doc); err != nil {
		return nil, err
	}

	command := doc.Command()

	dbName, err := getRequiredParam[string](doc, "$db")
	if err != nil {
		return nil, err
	}

	collection, err := getRequiredParam[string](doc, command)
	if err != nil {
		return nil, err
	}

	scale := float64(1)

	if scaleV := doc.Get("scale"); scaleV != nil {
		switch scaleV := scaleV.(type) {
		case float64:
			scale = scaleV
		case wirebson.NullType:
		case int32:
			scale = float64(scaleV)
		case int64:
			scale = float64(scaleV)
		default:
			// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/559
			msg := fmt.Sprintf(
				`BSON field 'collStats.scale' is the wrong type '%T', expected types '[long, int, decimal, double]'`,
				scaleV,
			)

			return nil, mongoerrors.NewWithArgument(mongoerrors.ErrTypeMismatch, msg, command)
		}
	}

	conn, err := h.Pool.Acquire()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}
	defer conn.Release()

	page, err := documentdb_api.CollStats(connCtx, conn.Conn(), h.L, dbName, collection, scale)
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	return middleware.MakeResponse(page)
}
