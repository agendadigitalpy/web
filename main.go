package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/badoux/checkmail"
	"github.com/gin-gonic/gin"
	strip "github.com/grokify/html-strip-tags-go"
)

const (
	defaultMongoURL = "mongodb://localhost:27017"
	defaultDBName   = "agendadigitalpy"
	proposalsCol    = "proposals"

	invalidEmailErr = "Dirección inválida"
	invalidCatErr   = "Categoría inválida"
	genericErr      = "Error"
)

var (
	db             *mgo.Database
	pubInfoTracker *PublicInfoTracker
)

func init() {
	pubInfoTracker = &PublicInfoTracker{}
	pubInfoTracker.FetchUpdates()
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		mongoURL = defaultMongoURL
	}
	sess, err := mgo.Dial(mongoURL)
	if err != nil {
		panic(err)
	}
	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = defaultDBName
	}
	db = sess.DB(dbName)
}

type Proposal struct {
	ID       bson.ObjectId `json:"_id" bson:"_id"`
	Name     string        `json:"name" bson:"name"`
	Email    string        `json:"email" bson:"email"`
	Category int           `json:"category" bson:"category"`
	Title    string        `json:"title" bson:"title"`
	Content  string        `json:"content" bson:"content"`
	Approved bool          `json:"approved" bson:"approved"`
}

type ProposalItem struct {
	ID               string
	Name             string
	Category         string
	CategoryLink     string
	Date             string
	TruncatedTitle   string
	Title            string
	TruncatedContent string
	Content          template.HTML
}

func storeProposal(p *Proposal) error {
	p.ID = bson.NewObjectId()
	p.Approved = true
	return db.C(proposalsCol).Insert(p)
}

func getProposal(id string) (i *ProposalItem) {
	p := Proposal{}
	q := db.C(proposalsCol).FindId(bson.ObjectIdHex(id))
	err := q.One(&p)
	if err != nil {
		return nil
	}
	return renderProposal(&p)
}

func renderProposal(p *Proposal) *ProposalItem {
	cat := "General"
	catLink := "#"
	switch p.Category {
	case 1:
		cat = "Gobierno Digital"
		catLink = "/gobierno-digital"
	case 2:
		cat = "Economía Digital"
		catLink = "/economia-digital"
	case 3:
		cat = "Conectividad"
		catLink = "/conectividad"
	case 4:
		cat = "Fortalecimiento Institucional"
		catLink = "/fortalecimiento-institucional"
	}
	ts := p.ID.Time()
	content := strings.Replace(p.Content, "\r", "", -1)
	content = strings.Replace(content, "\n", "<br />", -1)
	i := &ProposalItem{
		Category:         cat,
		CategoryLink:     catLink,
		ID:               p.ID.Hex(),
		Name:             p.Name,
		Date:             ts.Format("02/01/06"),
		TruncatedTitle:   truncate(p.Title),
		TruncatedContent: truncate(p.Content),
		Title:            p.Title,
		Content:          template.HTML(content),
	}
	return i
}

func getProposals() (results []*ProposalItem, err error) {
	proposals := []Proposal{}
	q := db.C(proposalsCol).Find(bson.M{"approved": true}).Sort("-_id")
	err = q.All(&proposals)
	if err != nil {
		return results, err
	}

	for _, p := range proposals {
		i := renderProposal(&p)
		results = append(results, i)
	}

	return results, err
}

func truncate(s string) string {
	if len(s) > 50 {
		return fmt.Sprintf("%.50s...", s)
	}
	return s
}

func main() {
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})
	router.GET("/gobierno-digital", func(c *gin.Context) {
		c.HTML(http.StatusOK, "gobierno-digital.html", gin.H{})
	})
	router.GET("/economia-digital", func(c *gin.Context) {
		c.HTML(http.StatusOK, "economia-digital.html", gin.H{})
	})
	router.GET("/conectividad", func(c *gin.Context) {
		c.HTML(http.StatusOK, "conectividad.html", gin.H{})
	})
	router.GET("/fortalecimiento-institucional", func(c *gin.Context) {
		c.HTML(http.StatusOK, "fortalecimiento-institucional.html", gin.H{})
	})
	router.GET("/propuestas", func(c *gin.Context) {
		proposals, _ := getProposals()
		c.HTML(http.StatusOK, "propuestas.html", gin.H{
			"proposals": proposals,
		})
	})
	router.POST("/propuestas", func(c *gin.Context) {
		proposals, _ := getProposals()

		name := c.PostForm("name")
		name = strip.StripTags(name)
		email := c.PostForm("email")
		email = strip.StripTags(email)

		if err := checkmail.ValidateFormat(email); err != nil {
			c.HTML(http.StatusOK, "propuestas.html", gin.H{
				"proposals": proposals,
				"error":     true,
				"message":   invalidEmailErr,
			})
			return
		}

		category := c.PostForm("category")
		cat, err := strconv.Atoi(category)
		if err != nil {
			c.HTML(http.StatusOK, "propuestas.html", gin.H{
				"proposals": proposals,
				"error":     true,
				"message":   invalidCatErr,
			})
			return
		}
		if cat > 4 || cat < 0 {
			c.HTML(http.StatusOK, "propuestas.html", gin.H{
				"proposals": proposals,
				"error":     true,
				"message":   invalidCatErr,
			})
			return
		}
		title := c.PostForm("title")
		title = strip.StripTags(title)
		content := c.PostForm("content")
		content = strip.StripTags(content)
		p := &Proposal{
			Name:     name,
			Email:    email,
			Title:    title,
			Category: cat,
			Content:  content,
		}
		err = storeProposal(p)
		if err != nil {
			c.HTML(http.StatusOK, "propuestas.html", gin.H{
				"proposals": proposals,
				"error":     true,
				"message":   genericErr,
			})
			return
		}

		proposals, _ = getProposals()

		c.HTML(http.StatusOK, "propuestas.html", gin.H{
			"proposals": proposals,
			"success":   true,
		})
	})
	router.GET("/propuestas/:id", func(c *gin.Context) {
		id := c.Param("id")
		p := getProposal(id)
		c.HTML(http.StatusOK, "propuesta.html", gin.H{
			"p": p,
		})
	})

	router.GET("/documentacion", func(c *gin.Context) {
		c.HTML(http.StatusOK, "documentacion.html", gin.H{})
	})

	router.GET("/seguimiento", func(c *gin.Context) {
		c.HTML(http.StatusOK, "seguimiento.html", gin.H{})
	})

	router.GET("/solicitudes", func(c *gin.Context) {
		c.HTML(http.StatusOK, "solicitudes.html", gin.H{
			"reqs": pubInfoTracker.Reqs,
		})
	})

	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "error.html", gin.H{})
	})
	router.Run("localhost:8080")
}
