import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from './contexts/Auth'
import Layout from './Layout'
import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import Upload from './pages/Upload'
import PhotoDetail from './pages/PhotoDetail'
import Favorites from './pages/Favorites'

const queryClient = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Home />} />
              <Route path="login" element={<Login />} />
              <Route path="register" element={<Register />} />
              <Route path="upload" element={<Upload />} />
              <Route path="photo/:id" element={<PhotoDetail />} />
              <Route path="favorites" element={<Favorites />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  )
}
