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

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/FerretDB/FerretDB/v2/integration/setup"
)

func TestLogoutCommand(t *testing.T) {
	t.Parallel()

	s := setup.SetupWithOpts(t, nil)
	ctx, db := s.Ctx, s.Collection.Database()

	opts := options.Client().ApplyURI(s.MongoDBURI)
	client, err := mongo.Connect(ctx, opts)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(ctx))
	})

	db = client.Database(db.Name())

	var res bson.D
	err = db.RunCommand(ctx, bson.D{{"logout", 1}}).Decode(&res)
	require.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, res)

	// the test user logs out again, it has no effect
	err = db.RunCommand(ctx, bson.D{{"logout", 1}}).Decode(&res)
	require.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, res)
}

func TestLogoutCommandAuthenticatedUser(tt *testing.T) {
	t := setup.FailsForFerretDB(tt, "https://github.com/FerretDB/FerretDB-DocumentDB/issues/953")

	tt.Parallel()

	s := setup.SetupWithOpts(tt, nil)
	ctx, db := s.Ctx, s.Collection.Database()
	username, password, mechanism := "logoutuser", "testpass", "SCRAM-SHA-256"

	// TODO https://github.com/FerretDB/FerretDB-DocumentDB/issues/864
	_ = db.RunCommand(ctx, bson.D{{"dropUser", username}})

	err := db.RunCommand(ctx, bson.D{
		{"createUser", username},
		{"roles", bson.A{}},
		{"pwd", password},
		{"mechanisms", bson.A{mechanism}},
	}).Err()
	require.NoError(t, err, "cannot create user")

	credential := options.Credential{
		AuthMechanism: mechanism,
		AuthSource:    db.Name(),
		Username:      username,
		Password:      password,
	}

	opts := options.Client().ApplyURI(s.MongoDBURI).SetAuth(credential)
	client, err := mongo.Connect(ctx, opts)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(ctx))
	})

	db = client.Database(db.Name())

	var res bson.D
	err = db.RunCommand(ctx, bson.D{{"connectionStatus", 1}}).Decode(&res)
	require.NoError(t, err)

	expected := bson.D{
		{"authInfo", bson.D{
			{"authenticatedUsers", bson.A{bson.D{{"user", username}, {"db", db.Name()}}}},
			{"authenticatedUserRoles", bson.A{}},
		}},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, res)

	err = db.RunCommand(ctx, bson.D{{"logout", 1}}).Decode(&res)
	require.NoError(t, err)

	AssertEqualDocuments(t, bson.D{{"ok", float64(1)}}, res)

	err = db.RunCommand(ctx, bson.D{{"connectionStatus", 1}}).Decode(&res)
	require.NoError(t, err)

	expected = bson.D{
		{"authInfo", bson.D{
			{"authenticatedUsers", bson.A{}},
			{"authenticatedUserRoles", bson.A{}},
		}},
		{"ok", float64(1)},
	}

	AssertEqualDocuments(t, expected, res)

	_, err = db.Collection(s.Collection.Name()).InsertOne(ctx, bson.D{{"foo", "bar"}})

	// after logout FerretDB returns `(AuthenticationFailed) Authentication failed`
	// after logout MongoDB returns `(Unauthorized) Command insert requires authentication`
	// TODO https://github.com/FerretDB/FerretDB/issues/3974
	require.Error(t, err)
}
