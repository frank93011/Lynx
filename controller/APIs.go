package respond

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"Lynx/models"
	"Lynx/service"
	"Lynx/viewModels"

	uuid "github.com/nu7hatch/gouuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Login(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var user models.User
	var response = models.Success{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	userResult, err := service.GetUser(database, user.ToQueryBson())
	// if no user found then insert a new one and return
	if userResult == nil {
		_, err := service.SaveUser(database, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		// check if insert user
		_, err = service.GetUser(database, user.ToQueryBson())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		response.Success = true
		response.Message = "Add New User"
		jsondata, _ := json.Marshal(response)
		w.Write(jsondata)
		return nil
	}
	// if already has users
	response.Success = true
	response.Message = "User Login"
	jsondata, _ := json.Marshal(response)
	w.Write(jsondata)
	return nil
}

func GetUsersByProject(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	projectId, _ := strconv.Atoi(queryInfo["projectId"])
	var queryAuth = models.Auth{ProjectId: projectId}
	auths, err := service.GetAuthByProjectId(database, queryAuth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	log.Println("Authes", auths)
	var userIds []string
	for _, auth := range auths {
		userIds = append(userIds, auth.UserId)
	}
	users, err := service.GetUsersByIds(database, userIds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	var result = []viewModels.ProjectManageViewModel{}
	for _, user := range users {
		var userAuth = models.Auth{}
		for _, auth := range auths {
			if auth.UserId == user.UserId {
				userAuth = auth
				break
			}
		}
		var userViewModel = viewModels.ProjectManageViewModel{
			ProjectId:  userAuth.ProjectId,
			UserId:     userAuth.UserId,
			CodeType:   userAuth.CodeType,
			StatusCode: userAuth.StatusCode,
			Name:       user.Name,
			Email:      user.Email,
			ImageUrl:   user.ImageUrl,
		}
		result = append(result, userViewModel)
	}

	jsondata, _ := json.Marshal(result)
	w.Write(jsondata)
	return nil
}

func SaveAuth(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var requestBody models.Auth
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	requestBody.CodeType = "1"
	res, err := service.SaveAuth(database, requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	var response = models.Success{
		Success: true,
		Message: res.InsertedID.(primitive.ObjectID).Hex(),
	}
	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

func SaveProject(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var addProjModel = viewModels.AddProjectViewModel{}
	err := json.NewDecoder(r.Body).Decode(&addProjModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// assign new projectId to each member's authentication
	currentId, err := service.GetProjectCount(database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// check if csv format decode is correct first
	articles, tasks, err := convertToArticlesAndTasks(addProjModel.CsvFile, int(currentId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// then adding objects into db
	addProjModel.Project.ProjectId = int(currentId)
	projectResult, err := service.SaveProject(database, addProjModel.Project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	for i := range addProjModel.Members {
		addProjModel.Members[i].ProjectId = int(currentId)
	}
	log.Println("addProj members", addProjModel.Members)
	_, err = service.SaveAuths(database, addProjModel.Members)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	_, err = service.SaveArticles(database, articles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	_, err = service.SaveTasks(database, tasks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	var response = models.Success{
		Success: true,
		Message: projectResult.InsertedID.(primitive.ObjectID).Hex(),
	}
	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

// private func for decode articles and Tasks.
func convertToArticlesAndTasks(csvFile [][]string, currentId int) ([]models.Article, []models.MRCTask, error) {
	var articles []models.Article
	var tasks []models.MRCTask
	articleSets := make(map[string]int)
	var ai int = -1 //point index of articles
	for _, pair := range csvFile {
		var contexts = pair[0]
		var indices = strings.Split(pair[1], "-")
		var articleId = "articleId" + indices[0]
		var taskId = "taskId" + pair[1]
		// basic check for the csv format
		if len(pair) != 2 || len(indices) != 2 {
			log.Println("錯誤的CSV檔案格式")
			return nil, nil, errors.New("錯誤的CSV檔案格式")
		}
		// check if the articles already exist else create one
		if _, ok := articleSets[articleId]; ok {
			articles[ai].TotalTasks += 1
		} else {
			articleSets[articleId] = 1
			var article = models.Article{
				ArticleId:    articleId,
				ProjectId:    currentId,
				ArticleTitle: contexts,
				TotalTasks:   1,
			}
			articles = append(articles, article)
			ai += 1
		}
		var task = models.MRCTask{
			TaskId:    taskId,
			ArticleId: articleId,
			TaskType:  "MRC",
			TaskTitle: contexts,
			Context:   contexts,
		}
		tasks = append(tasks, task)
	}
	return articles, tasks, nil
}

func GetUsers(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	users, err := service.GetUsers(database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	jsondata, _ := json.Marshal(users)
	w.Write(jsondata)
	return nil
}

func GetProjects(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	var userId = queryInfo["userId"]
	var queryAuth = models.Auth{UserId: userId, StatusCode: "1"}
	log.Println(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	projects, err := service.GetProjectsByAuth(database, queryAuth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	jsondata, _ := json.Marshal(projects)
	w.Write(jsondata)
	return nil
}

func GetArticles(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	var result viewModels.ProjectViewModel
	var articles []models.Article
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	var userId = queryInfo["userId"]
	log.Println(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	//[BUG]
	var queryProject = models.Project{ProjectId: 1}
	projectResult, err := service.GetProjectByProjectId(database, queryProject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	articles, err = service.GetArticles(database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	result.ProjectId = projectResult.ProjectId
	result.ProjectName = projectResult.ProjectName
	result.ProjectType = projectResult.ProjectType
	result.LabelInfo = projectResult.LabelInfo
	result.ArticleList = articles
	// [PENDING] Further information for articles
	// for i, article := range articles {
	// 	// get how many tasks that each article has
	// 	tasks, err := service.GetTasksByArticleId(database, article.ToQueryBson())
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusBadRequest)
	// 		return err
	// 	}
	// 	articles[i].TotalTasks = len(tasks)
	// 	for _, task := range tasks {
	// 		answers, err := service.GetAnswers(database, models.MRCAnswer{UserId: userId, TaskId: task.TaskId})
	// 		if err != nil {
	// 			http.Error(w, err.Error(), http.StatusBadRequest)
	// 			return err
	// 		}
	// 		// get how many tasks user has answered
	// 		articles[i].Answered = len(answers)
	// 	}
	// }
	jsondata, _ := json.Marshal(result)
	w.Write(jsondata)
	return nil
}

func GetTasksByArticleId(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	var result viewModels.TasksViewModel
	// decode request condition to queryInfo
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	log.Println("GetTasksByArticleId queryInfo:", queryInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	// get tasks by articles
	tasks, err := service.GetTasksByArticleId(database, bson.M{"articleId": queryInfo["articleId"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// get ArticleInfo
	result.ArticleId = queryInfo["ArticleId"]
	result.TaskType = queryInfo["TaskType"]
	articleResult, err := service.GetArticleByArticleId(database, bson.M{"articleId": queryInfo["articleId"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	result.ArticleTitle = articleResult.ArticleTitle

	// get tasksInfo
	for _, task := range tasks {
		var t = viewModels.TaskListModel{
			TaskId:    task.TaskId,
			TaskTitle: task.TaskTitle,
			Context:   task.Context,
		}
		answers, err := service.GetAnswers(database, models.MRCAnswer{UserId: queryInfo["userId"], ArticleId: queryInfo["articleId"], TaskId: task.TaskId})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		t.Answered = len(answers)
		// if user has answers the question
		if len(answers) != 0 {
			t.IsAnswered = true
		}
		result.TaskList = append(result.TaskList, t)
	}
	jsondata, _ := json.Marshal(result)
	w.Write(jsondata)
	return nil
}

//[TODO] update MRCTask for answered + 1
func SaveArticles(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	collection := database.Collection("Articles")
	var articles []models.Article
	err := json.NewDecoder(r.Body).Decode(&articles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	for i, _ := range articles {
		articleId, _ := uuid.NewV4()
		articles[i].ArticleId = "articleId" + articleId.String()
	}
	log.Println(articles)

	// to insert into db, need to convert struct to interface{}
	docs := make([]interface{}, len(articles))
	for i, u := range articles {
		docs[i] = u
	}
	articleResult, err := collection.InsertMany(context.Background(), docs)
	if err != nil {
		log.Fatal(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	log.Printf("Inserted %v documents into articles collection!\n", len(articleResult.InsertedIDs))
	jsondata, _ := json.Marshal(models.InsertSuccess)
	_, _ = w.Write(jsondata)
	return nil
}

func GetTaskById(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	log.Println("GetTaskById queryInfo:", requestBody)
	answers, err := service.GetAnswers(database, models.MRCAnswer{UserId: requestBody["userId"], ArticleId: requestBody["articleId"], TaskId: requestBody["taskId"], TaskType: requestBody["taskType"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	task, err := service.GetTaskById(database, models.MRCTask{ArticleId: requestBody["articleId"], TaskId: requestBody["taskId"], TaskType: requestBody["taskType"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	var response viewModels.TaskViewModel
	response.TaskId = task.TaskId
	response.TaskType = task.TaskType
	response.TaskTitle = task.TaskTitle
	response.Answered = task.Answered
	response.Context = task.Context

	for _, answer := range answers {
		var QAPair = viewModels.QAPairModel{
			Question: answer.Question,
			Answer:   answer.Answer,
		}
		response.QAPairs = append(response.QAPairs, QAPair)
	}

	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

func SaveAnswer(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var requestBody models.MRCAnswer
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	res, err := service.SaveAnswer(database, requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	var response = models.Success{
		Success: true,
		Message: res.InsertedID.(primitive.ObjectID).Hex(),
	}
	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

func Test(w http.ResponseWriter, r *http.Request) error {
	var requestModel []interface{}
	err := json.NewDecoder(r.Body).Decode(&requestModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	log.Println(requestModel)
	jsondata, _ := json.Marshal(models.InsertSuccess)
	_, _ = w.Write(jsondata)
	return nil
}

//================================= sentiment API =================================
func GetSentiArticles(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	var articles []models.Article
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	var userId = queryInfo["userId"]
	log.Println(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	articles, err = service.GetSentiArticles(database)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	jsondata, _ := json.Marshal(articles)
	w.Write(jsondata)
	return nil
}

func GetSentiTasksByArticleId(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var queryInfo map[string]string
	var result viewModels.SentiTasksViewModel
	// decode request condition to queryInfo
	err := json.NewDecoder(r.Body).Decode(&queryInfo)
	log.Println("GetSentiTasksByArticleId queryInfo:", queryInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	// get tasks by articles
	tasks, err := service.GetSentiTasksByArticleId(database, bson.M{"articleId": queryInfo["articleId"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// get ArticleInfo
	result.ArticleId = queryInfo["articleId"]
	result.TaskType = queryInfo["taskType"]
	articleResult, err := service.GetSentiArticleByArticleId(database, bson.M{"articleId": queryInfo["articleId"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	result.ArticleTitle = articleResult.ArticleTitle

	// get tasksInfo
	for _, task := range tasks {
		var t = viewModels.SentiTaskListModel{
			TaskId:     task.TaskId,
			TaskTitle:  task.TaskTitle,
			Context:    task.Context,
			AspectPool: task.AspectPool,
		}

		answers, err := service.GetAspectByTaskId(database, models.SentiAspect{TaskId: task.TaskId})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
		// if user has answers the question
		if len(answers) != 0 {
			t.IsAnswered = true
		}

		result.TaskList = append(result.TaskList, t)
	}
	jsondata, _ := json.Marshal(result)
	w.Write(jsondata)
	return nil
}

func GetSentiTaskById(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	log.Println("GetSentiTaskById queryInfo:", requestBody)

	aspects, err := service.GetAspectByTaskId(database, models.SentiAspect{TaskId: requestBody["taskId"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	task, err := service.GetSentiTaskById(database, models.SentiTask{ArticleId: requestBody["articleId"], TaskId: requestBody["taskId"], TaskType: requestBody["taskType"]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	var response viewModels.SentiTaskViewModel
	// var response viewModels.SentiTasksViewModel
	response.TaskId = task.TaskId
	response.TaskType = task.TaskType
	response.TaskTitle = task.TaskTitle
	response.Context = task.Context
	response.AspectPool = task.AspectPool
	if len(aspects) != 0 {
		response.IsAnswered = true
	}

	// for _, answer := range answers {
	// 	var QAPair = viewModels.QAPairModel{
	// 		Question: answer.Question,
	// 		Answer:   answer.Answer,
	// 	}
	// 	response.QAPairs = append(response.QAPairs, QAPair)
	// }

	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

func SaveSentiAnswer(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
	var requestBody models.SentiAnswer
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	res, err := service.SaveSentiAnswer(database, requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	var response = models.Success{
		Success: true,
		Message: res.InsertedID.(primitive.ObjectID).Hex(),
	}
	jsondata, _ := json.Marshal(response)
	_, _ = w.Write(jsondata)
	return nil
}

// validation 才會用到的

// func GetSentiTaskById(database *mongo.Database, w http.ResponseWriter, r *http.Request) error {
// 	var requestBody map[string]string
// 	err := json.NewDecoder(r.Body).Decode(&requestBody)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return err
// 	}
// 	log.Println("GetSentiTaskById queryInfo:", requestBody)

// 	aspects, err := service.GetAspectByTaskId(database, models.SentiAspect{TaskId: requestBody["taskId"]})
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return err
// 	}

// 	task, err := service.GetTaskById(database, models.MRCTask{ArticleId: requestBody["articleId"], TaskId: requestBody["taskId"], TaskType: requestBody["taskType"]})
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return err
// 	}
// 	var response viewModels.TaskViewModel
// 	// var response viewModels.SentiTasksViewModel
// 	response.TaskId = task.TaskId
// 	response.TaskType = task.TaskType
// 	response.TaskTitle = task.TaskTitle
// 	response.Answered = task.Answered
// 	response.Context = task.Context

// 	for _, answer := range answers {
// 		var QAPair = viewModels.QAPairModel{
// 			Question: answer.Question,
// 			Answer:   answer.Answer,
// 		}
// 		response.QAPairs = append(response.QAPairs, QAPair)
// 	}

// 	jsondata, _ := json.Marshal(response)
// 	_, _ = w.Write(jsondata)
// 	return nil
// }
