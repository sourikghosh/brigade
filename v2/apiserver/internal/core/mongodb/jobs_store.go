package mongodb

import (
	"context"
	"fmt"

	"github.com/brigadecore/brigade/v2/apiserver/internal/core"
	"github.com/brigadecore/brigade/v2/apiserver/internal/lib/mongodb"
	"github.com/brigadecore/brigade/v2/apiserver/internal/meta"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// jobsStore is a MongoDB-based implementation of the core.JobsStore interface.
type jobsStore struct {
	collection mongodb.Collection
}

// NewJobsStore returns a MongoDB-based implementation of the core.JobsStore
// interface.
func NewJobsStore(database *mongo.Database) (core.JobsStore, error) {
	return &jobsStore{
		collection: database.Collection("events"),
	}, nil
}

func (j *jobsStore) Create(
	ctx context.Context,
	eventID string,
	jobName string,
	job core.Job,
) error {
	res, err := j.collection.UpdateOne(
		ctx,
		bson.M{"id": eventID},
		bson.M{
			"$set": bson.M{
				fmt.Sprintf("worker.jobs.%s", jobName): job,
			},
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error updating spec of event %q job %q",
			eventID,
			jobName,
		)
	}
	if res.MatchedCount == 0 {
		return &meta.ErrNotFound{
			Type: "Event",
			ID:   eventID,
		}
	}
	return nil
}

func (j *jobsStore) UpdateStatus(
	ctx context.Context,
	eventID string,
	jobName string,
	status core.JobStatus,
) error {
	res, err := j.collection.UpdateOne(
		ctx,
		bson.M{
			"id": eventID,
		},
		bson.M{
			"$set": bson.M{
				fmt.Sprintf("worker.jobs.%s.status", jobName): status,
			},
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error updating status of event %q worker job %q",
			eventID,
			jobName,
		)
	}
	if res.MatchedCount == 0 {
		return &meta.ErrNotFound{
			Type: "Job",
			ID:   eventID,
		}
	}
	return nil
}
