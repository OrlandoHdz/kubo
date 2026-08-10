package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type PermisoHandler struct {
	queries *db.Queries
}

func NewPermisoHandler(q *db.Queries) *PermisoHandler {
	return &PermisoHandler{queries: q}
}

// ListarPermisos devuelve el catálogo completo de módulos/permisos del menú.
// GET /api/v1/permisos
func (h *PermisoHandler) ListarPermisos(c *gin.Context) {
	permisos, err := h.queries.ListarPermisos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permisos)
}

// ListarPermisosUsuario devuelve los módulos con su estado (activo/inactivo) para un usuario.
// GET /api/v1/permisos/usuario/:id
func (h *PermisoHandler) ListarPermisosUsuario(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "ID de usuario inválido"})
		return
	}

	permisos, err := h.queries.ListarPermisosDeUsuario(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permisos)
}

// ActualizarPermisosUsuario activa/desactiva los módulos del menú para un usuario.
// PUT /api/v1/permisos/usuario/:id
// Body: { "permisos": [{ "clave": "dashboard", "activo": true }, ...] }
func (h *PermisoHandler) ActualizarPermisosUsuario(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "ID de usuario inválido"})
		return
	}

	var input struct {
		Permisos []struct {
			Clave  string `json:"clave"`
			Activo bool   `json:"activo"`
		} `json:"permisos"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	// Catálogo de módulos (clave -> id)
	catalogo, err := h.queries.ListarPermisos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	byClave := make(map[string]int32, len(catalogo))
	for _, p := range catalogo {
		byClave[p.Clave] = p.ID
	}

	// Quién realiza el cambio (desde el JWT)
	updatedBy := pgtype.Int4{Valid: false}
	if uid, ok := c.Get("userID"); ok {
		updatedBy = pgtype.Int4{Int32: uid.(int32), Valid: true}
	}

	for _, item := range input.Permisos {
		permisoID, ok := byClave[item.Clave]
		if !ok {
			continue
		}
		if err := h.queries.ActualizarPermisoUsuario(c.Request.Context(), db.ActualizarPermisoUsuarioParams{
			UsuarioID: int32(id),
			PermisoID: permisoID,
			Activo:    item.Activo,
			UpdatedBy: updatedBy,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"message": "Permisos actualizados correctamente"})
}
