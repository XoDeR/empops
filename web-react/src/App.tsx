import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import CompaniesPage from '@/pages/CompaniesPage'
import EmployeesPage from '@/pages/EmployeesPage'
import AdminlandPage from '@/pages/AdminlandPage'
import ProtectedRoute from '@/routes/ProtectedRoute'
import AppLayout from '@/routes/AppLayout'
import CompanyLayout from '@/routes/CompanyLayout'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<CompaniesPage />} />
          <Route path="/companies/:companyId" element={<CompanyLayout />}>
            <Route path="employees" element={<EmployeesPage />} />
            <Route path="adminland" element={<AdminlandPage />} />
          </Route>
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
