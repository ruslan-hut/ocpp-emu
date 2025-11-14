import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Stations from './pages/Stations'
import Messages from './pages/Messages'
import MessageCrafter from './pages/MessageCrafter'
import './App.css'

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/stations" element={<Stations />} />
          <Route path="/messages" element={<Messages />} />
          <Route path="/message-crafter" element={<MessageCrafter />} />
        </Routes>
      </Layout>
    </Router>
  )
}

export default App
