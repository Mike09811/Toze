// Package tests 端到端接口测试：走完整路由 + 内存数据库。
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/hash"
	"github.com/Mike09811/Toze/server/internal/testutil"
)

type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func request(t *testing.T, engine *gin.Engine, method, path string, body any, token string) (*httptest.ResponseRecorder, apiResp) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp apiResp
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w, resp
}

func login(t *testing.T, engine *gin.Engine, user, pass string) (string, apiResp) {
	t.Helper()
	_, resp := request(t, engine, http.MethodPost, "/api/login", map[string]string{
		"username": user, "password": pass,
	}, "")
	var data struct {
		Token string `json:"token"`
		UUID  string `json:"uuid"`
	}
	_ = json.Unmarshal(resp.Data, &data)
	return data.Token, resp
}

func TestHealth(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)

	w, _ := request(t, engine, http.MethodGet, "/healthz", nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("health status %d", w.Code)
	}
}

func TestLoginFlow(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)

	_, resp := login(t, engine, "admin", "wrong")
	if resp.Code != 2001 {
		t.Fatalf("want code 2001, got %d (%s)", resp.Code, resp.Msg)
	}

	token, resp := login(t, engine, "admin", "admin")
	if resp.Code != 0 || token == "" {
		t.Fatalf("login failed: %d %s", resp.Code, resp.Msg)
	}

	// 未携带 token
	w, _ := request(t, engine, http.MethodPost, "/api/system/user/list", map[string]any{}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}

	// 携带 token
	w, resp = request(t, engine, http.MethodPost, "/api/system/user/list",
		map[string]any{"offset": 0, "limit": 10, "date": []string{}}, token)
	if w.Code != http.StatusOK || resp.Code != 0 {
		t.Fatalf("list failed: %d %d %s", w.Code, resp.Code, resp.Msg)
	}
	var page struct {
		List  []map[string]any `json:"list"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.Total < 1 || len(page.List) < 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestUserCRUD(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	// 创建
	w, resp := request(t, engine, http.MethodPost, "/api/system/user/create", map[string]any{
		"name": "接口用户", "account": "apiuser", "password": "123456",
		"phone": "13800001111", "mail": "api@test.com", "gender": 1, "status": 1,
		"region": []string{"北京市", "市辖区", "东城区"}, "role": []string{"ROLE_USER"},
	}, token)
	if w.Code != 200 || resp.Code != 0 {
		t.Fatalf("create failed: %d %d %s", w.Code, resp.Code, resp.Msg)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(resp.Data, &created)
	if created.ID == "" {
		t.Fatal("created id is empty")
	}

	// 查询
	_, resp = request(t, engine, http.MethodPost, "/api/system/user/list", map[string]any{
		"account": "apiuser", "offset": 0, "limit": 10,
	}, token)
	var page struct {
		List  []map[string]any `json:"list"`
		Total int              `json:"total"`
	}
	_ = json.Unmarshal(resp.Data, &page)
	if page.Total != 1 {
		t.Fatalf("expect 1, got %d", page.Total)
	}

	// 更新
	_, resp = request(t, engine, http.MethodPost, "/api/system/user/update", map[string]any{
		"id": created.ID, "name": "接口用户改", "account": "apiuser", "phone": "13900002222",
	}, token)
	if resp.Code != 0 {
		t.Fatalf("update failed: %d %s", resp.Code, resp.Msg)
	}

	// 删除
	_, resp = request(t, engine, http.MethodPost, "/api/system/user/delete", map[string]any{"id": created.ID}, token)
	if resp.Code != 0 {
		t.Fatalf("delete failed: %d %s", resp.Code, resp.Msg)
	}

	// 再次查询应为空
	_, resp = request(t, engine, http.MethodPost, "/api/system/user/list", map[string]any{
		"account": "apiuser", "offset": 0, "limit": 10,
	}, token)
	_ = json.Unmarshal(resp.Data, &page)
	if page.Total != 0 {
		t.Fatalf("expect 0 after delete, got %d", page.Total)
	}
}

func TestRoleAndMenuEndpoints(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	_, resp := request(t, engine, http.MethodPost, "/api/system/role/list", map[string]any{"offset": 0, "limit": 10}, token)
	if resp.Code != 0 {
		t.Fatalf("role list: %d %s", resp.Code, resp.Msg)
	}

	_, resp = request(t, engine, http.MethodPost, "/api/system/menu/list", map[string]any{}, token)
	if resp.Code != 0 {
		t.Fatalf("menu list: %d %s", resp.Code, resp.Msg)
	}
	var menus struct {
		List   []map[string]any `json:"list"`
		Header []map[string]any `json:"header"`
	}
	_ = json.Unmarshal(resp.Data, &menus)
	if len(menus.List) == 0 {
		t.Fatal("menu list should not be empty")
	}

	_, resp = request(t, engine, http.MethodPost, "/api/system/i18n/list", map[string]any{}, token)
	if resp.Code != 0 {
		t.Fatalf("i18n list: %d %s", resp.Code, resp.Msg)
	}

	// 角色菜单分配
	_, resp = request(t, engine, http.MethodPost, "/api/system/role/menu/update", map[string]any{
		"id": "2", "menuIds": []string{"100", "101"},
	}, token)
	if resp.Code != 0 {
		t.Fatalf("role menu update: %d %s", resp.Code, resp.Msg)
	}
}

func TestPermissionGuard(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)

	// 造一个只有 ROLE_USER 的账号
	pwd, _ := hash.Make("user123")
	user := model.User{UUID: "u-1", Account: "normal", Password: pwd, Name: "普通用户", Status: 1, Gender: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	var role model.Role
	if err := db.Where("name = ?", "ROLE_USER").First(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
		t.Fatal(err)
	}

	engine, _ := testutil.NewRouter(t, db)
	token, resp := login(t, engine, "normal", "user123")
	if resp.Code != 0 {
		t.Fatalf("login: %d %s", resp.Code, resp.Msg)
	}

	// 普通用户可以读
	_, resp = request(t, engine, http.MethodPost, "/api/system/user/list", map[string]any{"offset": 0, "limit": 5}, token)
	if resp.Code != 0 {
		t.Fatalf("read should be allowed: %d %s", resp.Code, resp.Msg)
	}

	// 普通用户不能写
	w, _ := request(t, engine, http.MethodPost, "/api/system/user/create", map[string]any{
		"name": "x", "account": "hacker", "password": "123456",
	}, token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	_, resp := request(t, engine, http.MethodPost, "/api/logout", map[string]any{}, token)
	if resp.Code != 0 {
		t.Fatalf("logout: %d %s", resp.Code, resp.Msg)
	}

	w, _ := request(t, engine, http.MethodPost, "/api/system/user/list", map[string]any{}, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token should be rejected, got %d", w.Code)
	}
}

func TestBadRequest(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	req := httptest.NewRequest(http.MethodPost, "/api/system/user/create", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp apiResp
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 1001 {
		t.Fatalf("want 1001, got %d", resp.Code)
	}
}

func TestNotFoundAPI(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)

	w, _ := request(t, engine, http.MethodGet, "/api/nope", nil, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestAuditLogWritten(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	request(t, engine, http.MethodPost, "/api/system/user/create", map[string]any{
		"name": "审计", "account": "audituser", "password": "123456",
	}, token)

	var count int64
	db.Model(&model.OperationLog{}).Count(&count)
	if count == 0 {
		t.Fatal("operation log should be written")
	}
}

var _ = gorm.ErrRecordNotFound

func TestPaasEndpoints(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.Seed(t, db)
	engine, _ := testutil.NewRouter(t, db)
	token, _ := login(t, engine, "admin", "admin")

	// 新增服务器：响应不得回传凭证
	_, resp := request(t, engine, http.MethodPost, "/api/paas/server/create", map[string]any{
		"name": "vps-http", "ip": "203.0.113.20", "ssh_port": 22, "ssh_user": "root",
		"auth_type": 1, "auth_content": "secret-pass",
	}, token)
	if resp.Code != 0 {
		t.Fatalf("server create failed: %d %s", resp.Code, resp.Msg)
	}
	if bytes.Contains(resp.Data, []byte("secret-pass")) {
		t.Fatal("响应不能包含凭证明文")
	}
	var server struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(resp.Data, &server)
	if server.ID == "" {
		t.Fatal("server id is empty")
	}

	// 新增应用
	_, resp = request(t, engine, http.MethodPost, "/api/paas/app/create", map[string]any{
		"server_id": server.ID, "name": "web", "image": "nginx:1.27",
		"container_port": 80, "host_port": 8080, "restart": "always",
		"env": []string{"A=1"}, "volumes": []string{},
	}, token)
	if resp.Code != 0 {
		t.Fatalf("app create failed: %d %s", resp.Code, resp.Msg)
	}
	var app struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(resp.Data, &app)
	if app.ID == "" {
		t.Fatal("app id is empty")
	}

	// 非法应用名被拒
	_, resp = request(t, engine, http.MethodPost, "/api/paas/app/create", map[string]any{
		"server_id": server.ID, "name": "Bad!", "image": "nginx", "container_port": 80, "host_port": 8081,
	}, token)
	if resp.Code == 0 {
		t.Fatal("非法应用名应被拒绝")
	}

	// 绑定域名（stub SSH 下同步成功）
	_, resp = request(t, engine, http.MethodPost, "/api/paas/domain/create", map[string]any{
		"app_id": app.ID, "domain": "http-test.example.com",
	}, token)
	if resp.Code != 0 {
		t.Fatalf("domain create failed: %d %s", resp.Code, resp.Msg)
	}

	// 列表可见
	_, resp = request(t, engine, http.MethodPost, "/api/paas/domain/list", map[string]any{
		"offset": 0, "limit": 10,
	}, token)
	var page struct {
		List  []map[string]any `json:"list"`
		Total int              `json:"total"`
	}
	_ = json.Unmarshal(resp.Data, &page)
	if page.Total != 1 {
		t.Fatalf("expect 1 domain, got %d", page.Total)
	}

	// 无效 token 不能触发部署
	_, resp = request(t, engine, http.MethodPost, "/api/paas/app/deploy", map[string]any{"id": app.ID}, "invalid-token")
	if resp.Code == 0 {
		t.Fatal("无效 token 不应能调用部署")
	}
}
