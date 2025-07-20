package controllers

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webroot/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	UserID  int64  `json:"user_id"`
}

type LoginResponse struct {
	Message string `json:"message"`
}

type ForgotPasswordRequest struct {
	Username string `form:"username" binding:"required"`
	Email    string `form:"email" binding:"required"`
}

type bindmail struct {
	Email string `form:"email"`
}

type NavItem struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type LinksItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ArticleResponse struct {
    Pagination Pagination `json:"pagination"`
    Articles   []Article  `json:"articles"`
}

type Pagination struct {
    CurrentPage  int `json:"currentPage"`
    TotalPages   int `json:"totalPages"`
    ItemsPerPage int `json:"itemsPerPage"`
    TotalItems   int `json:"totalItems"`
}

type Article struct {
    ID      int    `json:"id"`
    Title   string `json:"title"`
    Meta    Meta   `json:"meta"`
    Image   string `json:"image"`
    Excerpt string `json:"excerpt"`
}

type Meta struct {
    Date     string `json:"date"`
    Category string `json:"category"`
    Views    int    `json:"views"`
}    

func BlogHandlers(c *gin.Context) {
	switch {
	case c.Request.URL.Path == "/":
		content, err := GetStaticFileContent("blog/index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	case c.Request.URL.Path == "/user":
		content, err := GetStaticFileContent("blog/user.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	case c.Request.URL.Path == "/test":
		content, err := GetStaticFileContent("blog/test.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	case c.Request.URL.Path == "/api":
		switch c.Request.Method {
		case http.MethodPost:
			handlePost(c)
		case http.MethodGet:
			handleGet(c)
		}
	default:
		c.JSON(http.StatusNotFound, gin.H{"msg": "目录不存在"})
	}
}

func handlePost(c *gin.Context) {
	action := c.Query("action")
	switch action {
	case "forgotPassword":
		handleForgotPassword_blog(c)
	case "register":
		handleRegister_blog(c)
	case "login":
		handleLogin_blog(c)
	case "bindEmail":
		bindMail(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "未知操作"})
	}
}

func handleGet(c *gin.Context) {
	action := c.Query("action")
	switch action {
	case "getArticles":
		getArticles_blog(c)
	case "getFriendLinks":
		getFriendLinks_blog(c)
	case "getNav":
		getNav_blog(c)
	case "checkLogin":
		handleCheckLogin_blog(c)
	case "captcha":
		c.Status(201)
		GetCaptcha(c)
	case "checkRToken":
		checkRToken(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "未知操作"})
	}
}

func getArticles_blog(c *gin.Context) {
    page := 1
    if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
        page = p
    }

    // 模拟数据库查询
    articles := []Article{
        {
            ID:    1,
            Title: "本站成立",
            Meta: Meta{
                Date:     "2025-07-13",
                Category: "公告",
                Views:    0,
            },
            Image:   "https://picsum.photos/400/240?random=1",
            Excerpt: "这一切只是一个测试的开始...",
        },
    }

    // 模拟分页计算
    totalItems := 1
    itemsPerPage := 1
    totalPages := (totalItems + itemsPerPage - 1) / itemsPerPage

    response := ArticleResponse{
        Pagination: Pagination{
            CurrentPage:  page,
            TotalPages:   totalPages,
            ItemsPerPage: itemsPerPage,
            TotalItems:   totalItems,
        },
        Articles: articles,
    }

    c.JSON(http.StatusOK, response)
}    
func getFriendLinks_blog(c *gin.Context) {
	linksData := []LinksItem{
		{Name: "测试", URL: "#"},
		{Name: "用户", URL: "/user"},
	}
	c.JSON(http.StatusOK, gin.H{"links": linksData})
}
func getNav_blog(c *gin.Context) {
	navData := []NavItem{
		{Text: "测试", URL: "#"},
		{Text: "用户", URL: "/user"},
	}
	c.JSON(http.StatusOK, gin.H{"data": navData})
}
func handleForgotPassword_blog(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}
	valid, err := VerifyForm(c)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码验证失败", "details": err.Error()})
		return
	}
	users, err := models.Query("user", map[string]interface{}{"Name": req.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(users) <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户不存在"})
		return
	}
	user := users[0]
	mail, ok := user["Email"].(string)
	if !ok || mail == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户未绑定邮箱，无法重置密码"})
		return
	}
	if req.Email != mail {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名与邮箱不匹配"})
		return
	}
	rToken := generateRToken()
	expirationTime := time.Now().Add(time.Hour * 1).Unix()
	host := c.Request.Host
	plan := fmt.Sprintf("您好，%s !\n\n您申请的密码重置链接如下：\n%s\n\n请注意，此链接将在60分钟后失效。请立即使用该链接重置您的密码。\n\n谢谢！", req.Username, fmt.Sprintf("http://%s/api?action=checkRToken&RToken=%s", host, rToken))
	err = SendEmail(mail, "逸的生活记录-密码重置", plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败", "details": err.Error()})
		return
	}
	userID := int64(user["UID"].(int64))
	_, err = models.Update("user",
		map[string]interface{}{"RToken": rToken, "RTExpiration": expirationTime},
		map[string]interface{}{"UID": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新RToken失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("重置密码邮件已发送%v", user["Email"])})
}

func checkRToken(c *gin.Context) {
	rToken := c.Query("RToken")
	if rToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少RToken"})
		return
	}
	users, err := models.Query("user", map[string]interface{}{"RToken": rToken})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(users) <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的RToken"})
		return
	}
	user := users[0]
	expirationTime, ok := user["RTExpiration"].(int64)
	if !ok || expirationTime < time.Now().Unix() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "RToken已过期"})
		return
	}
	mail, ok := user["Email"].(string)
	if !ok || mail == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户未绑定邮箱，无法重置密码"})
		return
	}
	userID := int64(user["UID"].(int64))
	password := MD5(generateRToken())
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	_, err = models.Update("user",
		map[string]interface{}{"Password": string(hashedPassword), "Token": "", "RToken": "", "RTExpiration": 0},
		map[string]interface{}{"UID": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败", "details": err.Error()})
		return
	}
	plan := fmt.Sprintf("您好，\n\n您的账户序号：%v\n用户名：%v\n密码已重置为：%v\n\n请使用新密码登录，并确保在首次登录后修改密码以增强安全性。\n\n谢谢！", userID, user["Name"], password)
	err = SendEmail(mail, "逸的生活记录-密码重置", plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "操作成功,新的密码已发送至邮箱"})
}
func bindMail(c *gin.Context) {
	session := sessions.Default(c)
	token := session.Get("token")
	var req bindmail
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}
	users, err := models.Query("user", map[string]interface{}{"Token": token})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(users) <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Token"})
		return
	}
	checkmail, err := models.Query("user", map[string]interface{}{"Email": req.Email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(checkmail) > 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "该邮箱已被绑定"})
		return
	}
	user := users[0]
	mail, ok := user["Email"].(string)
	if ok || mail != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户已绑定邮箱，无法重复绑定"})
		return
	}
	_, err = models.Update("user",
		map[string]interface{}{"Email": req.Email},
		map[string]interface{}{"Token": token})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新邮箱失败", "details": err.Error()})
		return
	}
	plan := fmt.Sprintf("您好，\n\n用户：%v\n您已绑定邮箱：%v\n\n您可以定期重置密码, 如遗忘密码请及时找回。\n\n谢谢！", user["Name"], req.Email)
	err = SendEmail(req.Email, "逸的生活记录-邮箱绑定", plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "操作成功"})
}
func handleRegister_blog(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}
	valid, err := VerifyForm(c)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码验证失败", "details": err.Error()})
		return
	}
	users, err := models.Query("user", map[string]interface{}{"Name": req.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(users) > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	userData := map[string]interface{}{
		"Name":     req.Username,
		"Password": string(hashedPassword),
	}
	userID, err := models.Insert("user", userData)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名或邮箱已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, RegisterResponse{
		Message: "注册成功",
		UserID:  userID,
	})
}

func handleLogin_blog(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}
	valid, err := VerifyForm(c)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码验证失败", "details": err.Error()})
		return
	}
	users, err := models.Query("user", map[string]interface{}{"Name": req.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "details": err.Error()})
		return
	}
	if len(users) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}
	user := users[0]
	storedHash, ok := user["Password"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码格式错误"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	userID := int64(user["UID"].(int64))
	token := generateToken(userID, req.Username)
	currentTime := time.Now().Unix()
	session := sessions.Default(c)
	session.Set("user_id", userID)
	session.Set("username", req.Username)
	session.Set("token", token)
	session.Set("last_login", currentTime)
	err = session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存会话失败", "details": err.Error()})
		return
	}
	_, err = models.Update("user",
		map[string]interface{}{"Token": token, "Time": currentTime},
		map[string]interface{}{"UID": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, LoginResponse{
		Message: "登录成功",
	})
}

func handleCheckLogin_blog(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	username := session.Get("username")
	token := session.Get("token")

	if userID == nil || username == nil || token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":       "缺少有效的 session",
			"authenticated": false,
		})
		return
	}

	users, err := models.Query("user", map[string]interface{}{"UID": userID, "Name": username})
	if err != nil || len(users) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":       "用户不存在或 session 失效",
			"authenticated": false,
		})
		return
	}
	user := users[0]

	storedToken, ok := user["Token"].(string)
	if !ok || storedToken != token {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":       "Token已失效",
			"authenticated": false,
		})
		return
	}
	lastLogin := session.Get("last_login")
	c.JSON(http.StatusOK, gin.H{
		"code":          200,
		"authenticated": true,
		"user": gin.H{
			"username": username,
			"user_id":  userID,
			"email":    user["Email"],
		},
		"lastLogin": lastLogin,
	})
}

func generateToken(userID int64, username string) string {
	return fmt.Sprintf("%d:%s:%x", userID, username, time.Now().UnixNano())
}

func generateRToken() string {
	return fmt.Sprintf("%x", time.Now().UnixNano()+rand.Int63())
}



func MD5(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}
