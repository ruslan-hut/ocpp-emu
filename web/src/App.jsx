import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import ProtectedRoute from './components/ProtectedRoute'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Stations from './pages/Stations'
import StationEdit from './pages/StationEdit'
import StationConfigPage from './pages/StationConfigPage'
import Messages from './pages/Messages'
import MessageCrafter from './pages/MessageCrafter'
import ScenarioRunner from './pages/ScenarioRunner'
import './App.css'

function App() {
  return (
    <Router>
      <Routes>
        {/* Public route */}
        <Route path="/login" element={<Login />} />

        {/* Protected routes */}
        <Route path="/*" element={
          <ProtectedRoute>
            <Layout>
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/stations" element={<Stations />} />
                <Route path="/stations/new" element={<StationEdit />} />
                <Route path="/stations/:id/edit" element={<StationEdit />} />
                <Route path="/stations/:id/config" element={<StationConfigPage />} />
                <Route path="/messages" element={<Messages />} />
                <Route path="/message-crafter" element={<MessageCrafter />} />
                <Route path="/scenarios" element={<ScenarioRunner />} />
              </Routes>
            </Layout>
          </ProtectedRoute>
        } />
      </Routes>
    </Router>
  )
}

export default App
