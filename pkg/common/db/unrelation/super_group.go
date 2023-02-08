package unrelation

import (
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/db/table/unrelation"
	"Open_IM/pkg/utils"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
)

type SuperGroupMongoDriver struct {
	MgoClient                  *mongo.Client
	MgoDB                      *mongo.Database
	superGroupCollection       *mongo.Collection
	userToSuperGroupCollection *mongo.Collection
}

func NewSuperGroupMongoDriver(mgoClient *mongo.Client) *SuperGroupMongoDriver {
	mgoDB := mgoClient.Database(config.Config.Mongo.DBDatabase)
	return &SuperGroupMongoDriver{MgoDB: mgoDB, MgoClient: mgoClient, superGroupCollection: mgoDB.Collection(unrelation.CSuperGroup), userToSuperGroupCollection: mgoDB.Collection(unrelation.CUserToSuperGroup)}
}

func (db *SuperGroupMongoDriver) CreateSuperGroup(sCtx mongo.SessionContext, groupID string, initMemberIDs []string) error {
	superGroup := unrelation.SuperGroupModel{
		GroupID:   groupID,
		MemberIDs: initMemberIDs,
	}
	_, err := db.superGroupCollection.InsertOne(sCtx, superGroup)
	if err != nil {
		return err
	}
	upsert := true
	opts := &options.UpdateOptions{
		Upsert: &upsert,
	}
	for _, userID := range initMemberIDs {
		_, err = db.userToSuperGroupCollection.UpdateOne(sCtx, bson.M{"user_id": userID}, bson.M{"$addToSet": bson.M{"group_id_list": groupID}}, opts)
		if err != nil {
			return err
		}
	}
	return nil

}

func (db *SuperGroupMongoDriver) GetSuperGroup(ctx context.Context, groupID string) (*unrelation.SuperGroupModel, error) {
	superGroup := unrelation.SuperGroupModel{}
	err := db.superGroupCollection.FindOne(ctx, bson.M{"group_id": groupID}).Decode(&superGroup)
	return &superGroup, err
}

func (db *SuperGroupMongoDriver) AddUserToSuperGroup(ctx context.Context, groupID string, userIDs []string) error {
	opts := options.Session().SetDefaultReadConcern(readconcern.Majority())
	return db.MgoDB.Client().UseSessionWithOptions(ctx, opts, func(sCtx mongo.SessionContext) error {
		_, err := db.superGroupCollection.UpdateOne(sCtx, bson.M{"group_id": groupID}, bson.M{"$addToSet": bson.M{"member_id_list": bson.M{"$each": userIDs}}})
		if err != nil {
			_ = sCtx.AbortTransaction(ctx)
			return err
		}
		upsert := true
		opts := &options.UpdateOptions{
			Upsert: &upsert,
		}
		for _, userID := range userIDs {
			_, err = db.userToSuperGroupCollection.UpdateOne(sCtx, bson.M{"user_id": userID}, bson.M{"$addToSet": bson.M{"group_id_list": groupID}}, opts)
			if err != nil {
				_ = sCtx.AbortTransaction(ctx)
				return utils.Wrap(err, "transaction failed")
			}
		}
		return sCtx.CommitTransaction(ctx)
	})
}

func (db *SuperGroupMongoDriver) RemoverUserFromSuperGroup(ctx context.Context, groupID string, userIDs []string) error {
	opts := options.Session().SetDefaultReadConcern(readconcern.Majority())
	return db.MgoDB.Client().UseSessionWithOptions(ctx, opts, func(sCtx mongo.SessionContext) error {
		_, err := db.superGroupCollection.UpdateOne(sCtx, bson.M{"group_id": groupID}, bson.M{"$pull": bson.M{"member_id_list": bson.M{"$in": userIDs}}})
		if err != nil {
			_ = sCtx.AbortTransaction(ctx)
			return err
		}
		err = db.RemoveGroupFromUser(sCtx, groupID, userIDs)
		if err != nil {
			_ = sCtx.AbortTransaction(ctx)
			return err
		}
		return sCtx.CommitTransaction(ctx)
	})
}

func (db *SuperGroupMongoDriver) GetSuperGroupByUserID(ctx context.Context, userID string) (*unrelation.UserToSuperGroupModel, error) {
	var user unrelation.UserToSuperGroupModel
	err := db.userToSuperGroupCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	return &user, utils.Wrap(err, "")
}

func (db *SuperGroupMongoDriver) GetJoinGroup(ctx context.Context, userID string, pageNumber, showNumber int32) (int32, []string, error) {
	//TODO implement me
	panic("implement me")
}

func (db *SuperGroupMongoDriver) MapGroupMemberCount(ctx context.Context, groupIDs []string) (map[string]uint32, error) {
	//TODO implement me
	panic("implement me")
}

func (db *SuperGroupMongoDriver) DeleteSuperGroup(ctx context.Context, groupID string) error {
	opts := options.Session().SetDefaultReadConcern(readconcern.Majority())
	return db.MgoDB.Client().UseSessionWithOptions(ctx, opts, func(sCtx mongo.SessionContext) error {
		superGroup := &unrelation.SuperGroupModel{}
		_, err := db.superGroupCollection.DeleteOne(sCtx, bson.M{"group_id": groupID})
		if err != nil {
			_ = sCtx.AbortTransaction(ctx)
			return err
		}
		if err = db.RemoveGroupFromUser(sCtx, groupID, superGroup.MemberIDs); err != nil {
			_ = sCtx.AbortTransaction(ctx)
			return err
		}
		return sCtx.CommitTransaction(ctx)
	})
}

func (db *SuperGroupMongoDriver) RemoveGroupFromUser(sCtx context.Context, groupID string, userIDs []string) error {
	_, err := db.userToSuperGroupCollection.UpdateOne(sCtx, bson.M{"user_id": bson.M{"$in": userIDs}}, bson.M{"$pull": bson.M{"group_id_list": groupID}})
	return err
}