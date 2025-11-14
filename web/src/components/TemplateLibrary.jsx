import { useState, useEffect } from 'react'
import { templateService } from '../services/templateService'
import './TemplateLibrary.css'

function TemplateLibrary({ onSelectTemplate, onClose, currentPayload, currentAction }) {
  const [templates, setTemplates] = useState({})
  const [selectedCategory, setSelectedCategory] = useState('Core')
  const [selectedTemplate, setSelectedTemplate] = useState(null)
  const [showSaveDialog, setShowSaveDialog] = useState(false)
  const [templateName, setTemplateName] = useState('')
  const [templateDescription, setTemplateDescription] = useState('')
  const [templateCategory, setTemplateCategory] = useState('Custom')

  useEffect(() => {
    loadTemplates()
  }, [])

  const loadTemplates = () => {
    const categorized = templateService.getTemplatesByCategory()
    setTemplates(categorized)

    // Select first available category
    const categories = Object.keys(categorized)
    if (categories.length > 0 && !categorized[selectedCategory]) {
      setSelectedCategory(categories[0])
    }
  }

  const handleSelectTemplate = (template) => {
    setSelectedTemplate(template)
  }

  const handleUseTemplate = () => {
    if (selectedTemplate && onSelectTemplate) {
      onSelectTemplate({
        action: selectedTemplate.action,
        payload: JSON.stringify(selectedTemplate.payload, null, 2)
      })
      onClose()
    }
  }

  const handleSaveAsTemplate = () => {
    setShowSaveDialog(true)
    setTemplateName('')
    setTemplateDescription('')
    setTemplateCategory('Custom')
  }

  const handleSaveTemplate = () => {
    if (!templateName.trim()) {
      alert('Please enter a template name')
      return
    }

    try {
      const payload = JSON.parse(currentPayload)
      const template = {
        name: templateName.trim(),
        description: templateDescription.trim(),
        category: templateCategory,
        action: currentAction,
        payload: payload
      }

      const success = templateService.saveTemplate(template)
      if (success) {
        setShowSaveDialog(false)
        loadTemplates()
        alert('Template saved successfully!')
      } else {
        alert('Failed to save template')
      }
    } catch (err) {
      alert(`Invalid JSON payload: ${err.message}`)
    }
  }

  const handleDeleteTemplate = (templateName) => {
    if (!confirm(`Delete template "${templateName}"?`)) {
      return
    }

    const success = templateService.deleteTemplate(templateName)
    if (success) {
      loadTemplates()
      if (selectedTemplate && selectedTemplate.name === templateName) {
        setSelectedTemplate(null)
      }
    } else {
      alert('Failed to delete template')
    }
  }

  const categories = Object.keys(templates).sort()
  const currentTemplates = templates[selectedCategory] || []

  return (
    <div className="template-library-overlay" onClick={onClose}>
      <div className="template-library-modal" onClick={(e) => e.stopPropagation()}>
        <div className="template-library-header">
          <h2>Message Template Library</h2>
          <div className="header-actions">
            {currentPayload && (
              <button
                className="btn-save-template"
                onClick={handleSaveAsTemplate}
              >
                💾 Save Current as Template
              </button>
            )}
            <button className="btn-close" onClick={onClose}>✕</button>
          </div>
        </div>

        <div className="template-library-body">
          {/* Category Sidebar */}
          <div className="template-categories">
            <h3>Categories</h3>
            <div className="category-list">
              {categories.map(category => (
                <button
                  key={category}
                  className={`category-btn ${selectedCategory === category ? 'active' : ''}`}
                  onClick={() => setSelectedCategory(category)}
                >
                  {category}
                  <span className="category-count">
                    {templates[category]?.length || 0}
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* Template List */}
          <div className="template-list">
            <h3>{selectedCategory} Templates</h3>
            {currentTemplates.length === 0 ? (
              <div className="empty-state">
                No templates in this category
              </div>
            ) : (
              <div className="templates-grid">
                {currentTemplates.map(template => (
                  <div
                    key={template.name}
                    className={`template-card ${selectedTemplate?.name === template.name ? 'selected' : ''}`}
                    onClick={() => handleSelectTemplate(template)}
                  >
                    <div className="template-card-header">
                      <h4>{template.name}</h4>
                      {!template.isBuiltIn && (
                        <button
                          className="btn-delete-small"
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDeleteTemplate(template.name)
                          }}
                          title="Delete template"
                        >
                          🗑️
                        </button>
                      )}
                    </div>
                    {template.description && (
                      <p className="template-description">{template.description}</p>
                    )}
                    <div className="template-meta">
                      <span className="template-action">{template.action}</span>
                      {template.isBuiltIn && (
                        <span className="badge-builtin">Built-in</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Template Preview */}
          <div className="template-preview">
            <h3>Preview</h3>
            {selectedTemplate ? (
              <>
                <div className="preview-info">
                  <div className="info-row">
                    <span className="label">Action:</span>
                    <span className="value">{selectedTemplate.action}</span>
                  </div>
                  {selectedTemplate.description && (
                    <div className="info-row">
                      <span className="label">Description:</span>
                      <span className="value">{selectedTemplate.description}</span>
                    </div>
                  )}
                </div>
                <div className="preview-payload">
                  <strong>Payload:</strong>
                  <pre>{JSON.stringify(selectedTemplate.payload, null, 2)}</pre>
                </div>
                <button
                  className="btn-use-template"
                  onClick={handleUseTemplate}
                >
                  📋 Use This Template
                </button>
              </>
            ) : (
              <div className="empty-state">
                Select a template to preview
              </div>
            )}
          </div>
        </div>

        {/* Save Template Dialog */}
        {showSaveDialog && (
          <div className="save-dialog-overlay" onClick={() => setShowSaveDialog(false)}>
            <div className="save-dialog" onClick={(e) => e.stopPropagation()}>
              <h3>Save as Template</h3>
              <div className="form-group">
                <label>Template Name *</label>
                <input
                  type="text"
                  value={templateName}
                  onChange={(e) => setTemplateName(e.target.value)}
                  placeholder="e.g., My Custom StartTransaction"
                  autoFocus
                />
              </div>
              <div className="form-group">
                <label>Description</label>
                <input
                  type="text"
                  value={templateDescription}
                  onChange={(e) => setTemplateDescription(e.target.value)}
                  placeholder="Optional description"
                />
              </div>
              <div className="form-group">
                <label>Category</label>
                <input
                  type="text"
                  value={templateCategory}
                  onChange={(e) => setTemplateCategory(e.target.value)}
                  placeholder="e.g., Custom, Testing"
                />
              </div>
              <div className="dialog-actions">
                <button className="btn-cancel" onClick={() => setShowSaveDialog(false)}>
                  Cancel
                </button>
                <button className="btn-save" onClick={handleSaveTemplate}>
                  Save Template
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default TemplateLibrary
