package service

import (
	"context"
	"log"
	"time"

	"github.com/Lynx/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetArticles(db *mongo.Database) ([]models.Article, error) {
	collection := db.Collection("Articles")
	var articles = []models.Article{}
	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.Article{}
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		articles = append(articles, result)
	}
	return articles, nil
}

// return only one article
func GetArticleByArticleId(db *mongo.Database, queryBson bson.M) (*models.Article, error) {
	ArticleCollection := db.Collection("Articles")
	var serviceResult models.Article
	cur := ArticleCollection.FindOne(context.Background(), queryBson)
	err := cur.Decode(&serviceResult)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return &serviceResult, nil
}

func GetTasksByArticleId(db *mongo.Database, queryBson bson.M) ([]models.MRCTask, error) {
	TaskCollection := db.Collection("MRCTask")
	var tasks []models.MRCTask
	cur, err := TaskCollection.Find(context.Background(), queryBson)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.MRCTask{}
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		tasks = append(tasks, result)
	}
	return tasks, nil
}

func GetAnswers(db *mongo.Database, task models.MRCAnswer) ([]*models.MRCAnswer, error) {
	AnswerCollection := db.Collection("MRCAnswer")
	var answers []*models.MRCAnswer
	cur, err := AnswerCollection.Find(context.Background(), task.ToQueryBson())
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.MRCAnswer{}
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		answers = append(answers, &result)
	}
	return answers, nil
}

func GetTaskById(db *mongo.Database, taskModel models.MRCTask) (*models.MRCTask, error) {
	TaskCollection := db.Collection("MRCTask")
	var task models.MRCTask
	result := TaskCollection.FindOne(context.Background(), taskModel.ToQueryBson())
	err := result.Decode(&task)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return &task, nil
}

func SaveAnswer(db *mongo.Database, answer models.MRCAnswer) (*mongo.InsertOneResult, error) {
	AnswerCollection := db.Collection("MRCAnswer")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := AnswerCollection.InsertOne(ctx, answer)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return res, nil
}
