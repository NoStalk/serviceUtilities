package serviceUtilities

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	platformDatapb "github.com/NoStalk/protoDefinitions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/**
* This is an interface to model the user schema
* It is used to model the stored data after querying the database.
**/

type UserSchema struct {
	Email        string
	FirstName    string
	LastName     string
	Password     string
	PlatformData Platforms
	RefreshToken string
}

type Platforms struct {
	Leetcode    PlatformDataModel
	Codeforces  PlatformDataModel
	Codechef    PlatformDataModel
	Cpoj        PlatformDataModel
	Hackerearth PlatformDataModel
	Atcoder     PlatformDataModel
}

type PlatformDataModel struct {
	Handle      string
	TotalSolved int32
	Ranking     float64
	Contests    []ContestData
	Submissions []SubmissionData
}

type ContestData struct {
	ContestName string
	ContestDate string
	Rank        float64
	Rating      float64
	Solved      int32
	ContestID   string
}
type SubmissionData struct {
	ProblemUrl         string
	ProblemName        string
	SubmissionDate     string
	SubmissionLanguage string
	SubmissionStatus   string
	CodeUrl            string
}

type DBResources struct {
	client             *mongo.Client
	ctx                context.Context
	cancel             context.CancelFunc
	selectedCollection *mongo.Collection
}

/**
* @brief: This function is used to create a new connection to the database.
* @param: None.
* @return: a mongo.Client object, a context object, and a contextCancel function.
**/

func OpenDatabaseConnection(mongoURI string) (DBResources, error) {

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))

	if err != nil {
		log.Printf("Couldnt connect to mongodb due to: %v", err)
		return DBResources{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var dbResources DBResources
	err = client.Connect(ctx)
	if err != nil {
		log.Printf("Cant connect to mongodb: %v", err)
		cancel()
		return dbResources, err
	}
	selectedCollection := client.Database("UserDB").Collection("users")
	fmt.Println("Connected to mongodb")
	dbResources = DBResources{
		client:             client,
		ctx:                ctx,
		cancel:             cancel,
		selectedCollection: selectedCollection,
	}
	return dbResources, nil
}

/**
* @brief: This function is used to get the last contest data of a user from the database.
* @param: email - the email of the user, platform - the platform of the user, dbResources - the database resources.
* @return: the last contest data of the user.
**/

func GetLastContest(email string, platform string, dbResources DBResources) ContestData {
	var documentResult bson.M
	filter := bson.M{
		"email": email,
	}
	opts := options.FindOne().SetProjection(bson.M{"platformData." + strings.ToLower(platform) + ".contests": 1})
	err := dbResources.selectedCollection.FindOne(dbResources.ctx, filter, opts).Decode(&documentResult)

	if err != nil {
		log.Fatalf("Couldnt find user: %v", err)
	}
	doc, err := bson.Marshal(documentResult)
	if err != nil {
		log.Fatalf("Couldnt marshal user: %v", err)
	}
	var userObject UserSchema
	err = bson.Unmarshal(doc, &userObject)
	if err != nil {
		log.Fatalf("Couldnt unmarshal user: %v", err)
	}

	platformData := getPlatformDataDynamically(&userObject.PlatformData, platform)

	if len(platformData.Contests) == 0 {
		var emptyContestDataStruct = ContestData{}
		return emptyContestDataStruct
	}
	return platformData.Contests[len(platformData.Contests)-1]
}

/**
* @brief: This function is used to get the last submission data of a user from the database.
* @param: email - the email of the user, platform - the platform of the user, dbResources - the database resources.
* @return: the last submission data of the user.
**/

func GetLastSubmission(email string, platform string, dbResources DBResources) SubmissionData {
	var documentResult bson.M
	filter := bson.M{
		"email": email,
	}
	opts := options.FindOne().SetProjection(bson.M{"platformData." + strings.ToLower(platform) + ".submissions": 1})
	err := dbResources.selectedCollection.FindOne(dbResources.ctx, filter, opts).Decode(&documentResult)

	if err != nil {
		log.Fatalf("Couldnt find user: %v", err)
	}
	doc, err := bson.Marshal(documentResult)
	if err != nil {
		log.Fatalf("Couldnt marshal user: %v", err)
	}
	var userObject UserSchema
	err = bson.Unmarshal(doc, &userObject)
	if err != nil {
		log.Fatalf("Couldnt unmarshal user: %v", err)
	}
	platformData := getPlatformDataDynamically(&userObject.PlatformData, platform)

	if len(platformData.Submissions) == 0 {
		var emptySubmissionDataStruct SubmissionData = SubmissionData{}
		return emptySubmissionDataStruct
	}

	return platformData.Submissions[len(platformData.Submissions)-1]

}

/**
* @brief: This function is used to find some user in the database and return user arrays.
* @param: *mongo.collection.
* @return: Array of contests objects and submissions objects.
* @deprecated: This function is deprecated.
* Deprecated: The function is no longer needed because dont query the entire arrays anymore!!
**/

func FindContestsandSubmissionsFromDB(dbResources DBResources, email string, platform string) ([]ContestData, []SubmissionData) {
	selectedCollection := dbResources.selectedCollection
	filter := bson.M{"email": email}
	var userMap map[string]interface{}
	var result bson.M
	err := selectedCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		log.Fatalf("Couldnt find user: %v", err)
	}
	doc, err := bson.Marshal(result)
	if err != nil {
		log.Fatalf("Couldnt marshal user: %v", err)
	}
	var userObject UserSchema
	err = bson.Unmarshal(doc, &userObject)
	if err != nil {
		log.Fatalf("Couldnt unmarshal user: %v", err)
	}
	err = bson.Unmarshal(doc, &userMap)

	if err != nil {
		log.Fatalf("Couldnt unmarshal user: %v", err)
	}

	platformData := getPlatformDataDynamically(&userObject.PlatformData, platform)
	return platformData.Contests, platformData.Submissions
}

/**
* @brief: This function is used to update the user's contest-data in the database.
* @param: *mongo.collection, user's email, array of contest-data.
* @return: None.
**/

func AppendContestData(dbResources DBResources, email string, platform string, newContestData []ContestData) error {
	selectedCollection := dbResources.selectedCollection
	// var updatedContests []ContestData = append(staleContestData, newContestData);
	updateContestQuery := bson.M{"$push": bson.M{"platformData." + strings.ToLower(platform) + ".contests": bson.M{"$each": newContestData}}}
	filter := bson.M{"email": email}
	// updatedUserSchemaDoc := bson.M{"$set": bson.M{"platformData.leetcode.contests": updatedContestQuery}};

	_, err := selectedCollection.UpdateOne(context.TODO(), filter, updateContestQuery)
	if err != nil {
		log.Fatalf("Couldnt update user: %v", err)
		return err
	}
	fmt.Println("Updated user")
	return nil
}

/**
* @brief: This function is used to update the user's submission-data in the database.
* @param: *mongo.collection, user's email, array of submission-data.
* @return: None.
**/

func AppendSubmissionData(dbResources DBResources, email string, platform string, newSubmissionData []SubmissionData) error {
	selectedCollection := dbResources.selectedCollection
	// var updatedSubmissions []SubmissionData = append(staleSubmissionData, newSubmissionData);
	updateSubmissionQuery := bson.M{"$push": bson.M{"platformData." + strings.ToLower(platform) + ".submissions": bson.M{"$each": newSubmissionData}}}
	filter := bson.M{"email": email}
	// updatedUserSchemaDoc := bson.M{"$set": bson.M{"platformData.leetcode.submissions": updatedSubmissionQuery}};

	_, err := selectedCollection.UpdateOne(context.TODO(), filter, updateSubmissionQuery)
	if err != nil {
		log.Fatalf("Couldnt update user: %v", err)
		return err
	}
	fmt.Println("Updated user")
	return nil
}

/**
* @brief: This function is used to format the data that was inserted into DB to the GRPC format that is sent as a response to the client.
* @param: The submission data that was inserted into DB.
* @return: SubmissionData array that is formatted to the GRPC format.
**/

func formatSubmissionSchemaToGRPCSubmission(submissionDataforDB []SubmissionData) []*platformDatapb.Submission {
	var grpcSubmissionData []*platformDatapb.Submission

	for _, submission := range submissionDataforDB {
		submissionResponseObject := platformDatapb.Submission{
			Date:          submission.SubmissionDate,
			Language:      submission.SubmissionLanguage,
			ProblemStatus: submission.SubmissionStatus,
			ProblemTitle:  submission.ProblemName,
			ProblemLink:   submission.ProblemUrl,
			CodeLink:      submission.CodeUrl,
		}
		grpcSubmissionData = append(grpcSubmissionData, &submissionResponseObject)

	}
	return grpcSubmissionData
}

/**
* @brief: This function is used to format the data that was inserted into DB to the GRPC format that is sent as a response to the client.
* @param: The submission data that was inserted into DB.
* @return: SubmissionData array that is formatted to the GRPC format.
**/

func formatContestSchemaToGRPCContest(contestDataforDB []ContestData) []*platformDatapb.Contest {
	var grpcContestData []*platformDatapb.Contest

	for _, contest := range contestDataforDB {
		contestIDParsedToFloat,err := strconv.ParseFloat(contest.ContestID, 64)
		if err != nil {
			log.Fatalf("Couldnt parse contestID to float: %v", err)
		}	
		contestResponseObject := platformDatapb.Contest{
			ContestName: contest.ContestName,
			Rank:        contest.Rank,
			Rating:      contest.Rating,
			ContestId:   contestIDParsedToFloat,
			ContestDate: contest.ContestDate,
		}
		grpcContestData = append(grpcContestData, &contestResponseObject)
	}
	return grpcContestData
}

func CreateGRPCSubmissionResponseFromSubmissionSchema(submissionDataforDB []SubmissionData) *platformDatapb.SubmissionResponse {
	response := &platformDatapb.SubmissionResponse{
		Submissions: formatSubmissionSchemaToGRPCSubmission(submissionDataforDB),
	}
	return response
}

func CreateGRPCContestResponseFromContestSchema(contestDataforDB []ContestData) *platformDatapb.ContestResponse {
	response := &platformDatapb.ContestResponse{
		Contests: formatContestSchemaToGRPCContest(contestDataforDB),
	}
	return response
}

func CreateGRPCCompleteUserDataResponseFromSubmssionSchema(submissionDataforDB []SubmissionData, contestDataforDB []ContestData) *platformDatapb.CompleteUserDataResponse {
	response := &platformDatapb.CompleteUserDataResponse{
		Submissions: formatSubmissionSchemaToGRPCSubmission(submissionDataforDB),
		Contests:    formatContestSchemaToGRPCContest(contestDataforDB),
	}
	return response
}

/**
* @brief: This function is used close the database connection.
* @param: *mongo.client, context, context cancel function.
* @return: None.
**/

func CloseDatabaseConnection(dbResources DBResources) {
	dbResources.client.Disconnect(dbResources.ctx)
	dbResources.cancel()
	fmt.Println("Disconnected from mongodb")
}

/**
* @brief: This function is used to dynamically get the platform data from the user object.
* @param: *UserSchema.Platforms, string.
* @return: PlatformDataModel.
**/

func getPlatformDataDynamically(platformData *Platforms, platform string) PlatformDataModel {
	reflectedValue := reflect.ValueOf(platformData).Elem()
	fieldValue := reflect.Indirect(reflectedValue).FieldByName(platform)
	return fieldValue.Interface().(PlatformDataModel)
}
