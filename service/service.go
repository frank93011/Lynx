package service

import (
	"context"
	"log"
	"time"

	"Lynx/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetAuthByProjectId(db *mongo.Database, auth models.Auth) ([]models.Auth, error) {
	collection := db.Collection(auth.TableName())
	var serviceResult = []models.Auth{}
	cur, err := collection.Find(context.Background(), bson.M{"projectId": auth.ProjectId})
	if err != nil {
		log.Println("Find Project Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.Auth{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Auth Error", err)
			return nil, err
		}
		serviceResult = append(serviceResult, result)
	}
	return serviceResult, nil
}

func GetProjectsByAuth(db *mongo.Database, auth models.Auth) ([]models.Project, error) {
	collection := db.Collection(auth.TableName())
	var authList = models.Auths{}
	cur, err := collection.Find(context.Background(), auth.ToQueryBson())
	log.Println("cur", cur)
	log.Println("auth", auth.ToQueryBson())
	if err != nil {
		log.Println("Find Project Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.Auth{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Project Error", err)
			return nil, err
		}
		authList = append(authList, result)
	}
	var projectIds = authList.SelectProjectIdList()
	var serviceResult = []models.Project{}
	for _, projectId := range projectIds {
		project, err := GetProjectByProjectId(db, models.Project{ProjectId: projectId})
		if err != nil {
			return nil, err
		}
		serviceResult = append(serviceResult, *project)
	}

	return serviceResult, nil
}

func SaveAuth(db *mongo.Database, auth models.Auth) (*mongo.InsertOneResult, error) {
	AuthCollection := db.Collection(auth.TableName())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := AuthCollection.InsertOne(ctx, auth)
	if err != nil {
		log.Println("Insert auth Error", err)
		return nil, err
	}
	return res, nil
}

func SaveAuths(db *mongo.Database, auths []models.Auth) (*mongo.InsertManyResult, error) {
	AuthCollection := db.Collection("Authentication")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// to insert into db, need to convert struct to interface{}
	insertDatas := make([]interface{}, len(auths))
	for i, a := range auths {
		insertDatas[i] = a
	}
	res, err := AuthCollection.InsertMany(ctx, insertDatas)
	if err != nil {
		log.Println("Insert auth Error", err)
		return nil, err
	}
	return res, nil
}

// save project articles
func SaveArticles(db *mongo.Database, articles []models.Article) (*mongo.InsertManyResult, error) {
	ArticleCollection := db.Collection("Articles")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// to insert into db, need to convert struct to interface{}
	insertDatas := make([]interface{}, len(articles))
	for i, a := range articles {
		insertDatas[i] = a
	}
	res, err := ArticleCollection.InsertMany(ctx, insertDatas)
	if err != nil {
		log.Println("Insert auth Error", err)
		return nil, err
	}
	return res, nil
}

func SaveTasks(db *mongo.Database, tasks []models.MRCTask) (*mongo.InsertManyResult, error) {
	TaskCollection := db.Collection("MRCTask")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// to insert into db, need to convert struct to interface{}
	insertDatas := make([]interface{}, len(tasks))
	for i, a := range tasks {
		insertDatas[i] = a
	}
	res, err := TaskCollection.InsertMany(ctx, insertDatas)
	if err != nil {
		log.Println("Insert auth Error", err)
		return nil, err
	}
	return res, nil
}

// func GetProjects(db *mongo.Database, userId string) ([]models.Project, error) {
// 	collection := db.Collection("Project")
// 	var serviceResult = []models.Project{}
// 	cur, err := collection.Find(context.Background(), bson.M{"manager": userId})
// 	if err != nil {
// 		log.Println("Find Project Error", err)
// 		return nil, err
// 	}
// 	for cur.Next(context.Background()) {
// 		result := models.Project{}
// 		err := cur.Decode(&result)
// 		if err != nil {
// 			log.Println("Decode Project Error", err)
// 			return nil, err
// 		}
// 		serviceResult = append(serviceResult, result)
// 	}
// 	return serviceResult, nil
// }

func GetProjectByProjectId(db *mongo.Database, project models.Project) (*models.Project, error) {
	collection := db.Collection("Project")
	var serviceResult = models.Project{}
	log.Println(project)
	cur := collection.FindOne(context.Background(), project.ToQueryBson())
	// if no project then return nil
	if cur.Err() != nil {
		log.Println("Can't find project in DB", cur.Err())
		return nil, cur.Err()
	}
	// if has project then return
	err := cur.Decode(&serviceResult)
	if err != nil {
		log.Println("Decode project Error", err)
		return nil, err
	}
	log.Println("Get project:", serviceResult)
	return &serviceResult, nil
}

func GetProjectCount(db *mongo.Database) (int64, error) {
	ProjectCollection := db.Collection("Project")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	itemCount, err := ProjectCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println("Insert project Error", err)
		return 0, err
	}
	return itemCount, nil
}

func SaveProject(db *mongo.Database, project models.Project) (*mongo.InsertOneResult, error) {
	ProjectCollection := db.Collection(project.TableName())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := ProjectCollection.InsertOne(ctx, project)
	if err != nil {
		log.Println("Insert project Error", err)
		return nil, err
	}
	return res, nil
}

func GetUsersByIds(db *mongo.Database, userIds []string) ([]models.User, error) {
	collection := db.Collection("GUser")
	var serviceResult = []models.User{}
	log.Println("userIds", userIds)
	// batch query from userIds
	cur, err := collection.Find(context.Background(), bson.M{"userId": bson.M{"$in": userIds}})
	if err != nil {
		log.Println("Find Users Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.User{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Users Error", err)
			return nil, err
		}
		serviceResult = append(serviceResult, result)
	}
	return serviceResult, nil
}

func GetUser(db *mongo.Database, queryBson bson.M) (*models.User, error) {
	collection := db.Collection("GUser")
	var serviceResult = models.User{}
	cur := collection.FindOne(context.Background(), queryBson)
	// if no user then return nil
	if cur.Err() != nil {
		log.Println("Can't find user in DB")
		return nil, cur.Err()
	}
	// if has user then return
	err := cur.Decode(&serviceResult)
	if err != nil {
		log.Println("Decode user Error", err)
		return nil, err
	}
	log.Println("Get user:", serviceResult)
	return &serviceResult, nil
}

func GetUsers(db *mongo.Database) ([]models.User, error) {
	collection := db.Collection("GUser")
	var serviceResult = []models.User{}
	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("Find user Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.User{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Article Error", err)
			return nil, err
		}
		serviceResult = append(serviceResult, result)
	}
	return serviceResult, nil
}

func SaveUser(db *mongo.Database, user models.User) (*mongo.InsertOneResult, error) {
	UserCollection := db.Collection("GUser")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := UserCollection.InsertOne(ctx, user)
	if err != nil {
		log.Println("Insert user Error", err)
		return nil, err
	}
	log.Println("Insert user Success", user)
	return res, nil
}

func GetArticlesByProjectId(db *mongo.Database, projectId int) ([]models.Article, error) {
	collection := db.Collection("Articles")
	var articles = []models.Article{}
	cur, err := collection.Find(context.Background(), bson.M{"projectId": projectId})
	if err != nil {
		log.Println("Find Articles Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.Article{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Article Error", err)
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
		log.Println("Decode articles Error", err)
		return nil, err
	}
	return &serviceResult, nil
}

func GetTasksByArticleId(db *mongo.Database, queryBson bson.M) ([]models.MRCTask, error) {
	TaskCollection := db.Collection("MRCTask")
	var tasks []models.MRCTask
	cur, err := TaskCollection.Find(context.Background(), queryBson)
	if err != nil {
		log.Println("Find tasks Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.MRCTask{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode tasks Error", err)
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
		log.Println("Find answers Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.MRCAnswer{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode answer Error", err)
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
		log.Println("Decode task Error", err)
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
		log.Println("Insert answers Error", err)
		return nil, err
	}
	return res, nil
}

//================================= sentiment API =================================
func GetSentiArticles(db *mongo.Database) ([]models.Article, error) {
	collection := db.Collection("SentiArticles")
	var articles = []models.Article{}
	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("Find Articles Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.Article{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode Article Error", err)
			return nil, err
		}
		articles = append(articles, result)
	}
	return articles, nil
}

func GetSentiArticleByArticleId(db *mongo.Database, queryBson bson.M) (*models.Article, error) {
	ArticleCollection := db.Collection("SentiArticles")
	var serviceResult models.Article
	cur := ArticleCollection.FindOne(context.Background(), queryBson)
	err := cur.Decode(&serviceResult)
	if err != nil {
		log.Println("Decode articles Error", err)
		return nil, err
	}
	return &serviceResult, nil
}

func GetSentiTasksByArticleId(db *mongo.Database, queryBson bson.M) ([]models.SentiTask, error) {
	TaskCollection := db.Collection("SentiTask")
	var tasks []models.SentiTask
	cur, err := TaskCollection.Find(context.Background(), queryBson)
	if err != nil {
		log.Println("Find tasks Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.SentiTask{}
		err := cur.Decode(&result)
		// log.Println(result)
		if err != nil {
			log.Println("Decode tasks Error", err)
			return nil, err
		}
		tasks = append(tasks, result)

	}
	return tasks, nil
}

func GetAspectByTaskId(db *mongo.Database, task models.SentiAspect) ([]*models.SentiAspect, error) {
	AnswerCollection := db.Collection("SentiAspect")
	var aspects []*models.SentiAspect
	cur, err := AnswerCollection.Find(context.Background(), task.ToQueryBson())
	if err != nil {
		log.Println("Find answers Error", err)
		return nil, err
	}
	for cur.Next(context.Background()) {
		result := models.SentiAspect{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println("Decode answer Error", err)
			return nil, err
		}
		aspects = append(aspects, &result)
	}
	return aspects, nil
}

func GetSentiTaskById(db *mongo.Database, taskModel models.SentiTask) (*models.SentiTask, error) {
	TaskCollection := db.Collection("SentiTask")
	var task models.SentiTask
	result := TaskCollection.FindOne(context.Background(), taskModel.ToQueryBson())
	err := result.Decode(&task)
	if err != nil {
		log.Println("Decode task Error", err)
		return nil, err
	}
	return &task, nil
}

func SaveSentiAnswer(db *mongo.Database, answer models.SentiAnswer) (*mongo.InsertOneResult, error) {
	AnswerCollection := db.Collection("SentiAnswer")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := AnswerCollection.InsertOne(ctx, answer)
	if err != nil {
		log.Println("Insert answers Error", err)
		return nil, err
	}
	return res, nil
}

//這邊要等到 validation 的時候才會用到
// func GetSentiAnswer(db *mongo.Database, task models.MRCAnswer) ([]*models.MRCAnswer, error) {
// 	AnswerCollection := db.Collection("MRCAnswer")
// 	var answers []*models.MRCAnswer
// 	cur, err := AnswerCollection.Find(context.Background(), task.ToQueryBson())
// 	if err != nil {
// 		log.Println("Find answers Error", err)
// 		return nil, err
// 	}
// 	for cur.Next(context.Background()) {
// 		result := models.MRCAnswer{}
// 		err := cur.Decode(&result)
// 		if err != nil {
// 			log.Println("Decode answer Error", err)
// 			return nil, err
// 		}
// 		answers = append(answers, &result)
// 	}
// 	return answers, nil
// }
