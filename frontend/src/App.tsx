import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from './contexts/Auth'
import { ThemeProvider } from './contexts/Theme'
import { ToastProvider } from './components/Toast'
import Layout from './Layout'
import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import Upload from './pages/Upload'
import PhotoDetail from './pages/PhotoDetail'
import Favorites from './pages/Favorites'
import Profile from './pages/Profile'
import EditPhoto from './pages/EditPhoto'
import Search from './pages/Search'
import Notifications from './pages/Notifications'
import Settings from './pages/Settings'

const queryClient = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <BrowserRouter>
              <Routes>
                <Route path="/" element={<Layout />}>
                  <Route index element={<Home />} />
                  <Route path="login" element={<Login />} />
                  <Route path="register" element={<Register />} />
                  <Route path="upload" element={<Upload />} />
                  <Route path="photo/:id" element={<PhotoDetail />} />
                  <Route path="photo/:id/edit" element={<EditPhoto />} />
                  <Route path="favorites" element={<Favorites />} />
                  <Route path="user/:username" element={<Profile />} />
                  <Route path="search" element={<Search />} />
                  <Route path="notifications" element={<Notifications />} />
                  <Route path="settings" element={<Settings />} />
                </Route>
              </Routes>
            </BrowserRouter>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}
