import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import CompaniesPage from '@/pages/CompaniesPage'
import EmployeesPage from '@/pages/EmployeesPage'
import EmployeeDetailPage from '@/pages/EmployeeDetailPage'
import AdminlandPage from '@/pages/AdminlandPage'
import TeamsPage from '@/pages/TeamsPage'
import TeamDetailPage from '@/pages/TeamDetailPage'
import ProjectsPage from '@/pages/ProjectsPage'
import ProjectDetailPage from '@/pages/ProjectDetailPage'
import RecruitingPage from '@/pages/RecruitingPage'
import JobOpeningDetailPage from '@/pages/JobOpeningDetailPage'
import OneOnOneDetailPage from '@/pages/OneOnOneDetailPage'
import DisciplineListPage from '@/pages/DisciplineListPage'
import DisciplineDetailPage from '@/pages/DisciplineDetailPage'
import SkillsPage from '@/pages/SkillsPage'
import MoraleHistoryPage from '@/pages/MoraleHistoryPage'
import HardwareListPage from '@/pages/HardwareListPage'
import HardwareDetailPage from '@/pages/HardwareDetailPage'
import SoftwareListPage from '@/pages/SoftwareListPage'
import SoftwareDetailPage from '@/pages/SoftwareDetailPage'
import JobsPage from '@/pages/public/JobsPage'
import CompanyJobsPage from '@/pages/public/CompanyJobsPage'
import JobApplyPage from '@/pages/public/JobApplyPage'
import DashboardPage from '@/pages/DashboardPage'
import ProtectedRoute from '@/routes/ProtectedRoute'
import AppLayout from '@/routes/AppLayout'
import CompanyLayout from '@/routes/CompanyLayout'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      <Route path="/jobs" element={<JobsPage />} />
      <Route path="/jobs/:companySlug" element={<CompanyJobsPage />} />
      <Route path="/jobs/:companySlug/jobs/:jobSlug" element={<JobApplyPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<CompaniesPage />} />
          <Route path="/companies/:companyId" element={<CompanyLayout />}>
            <Route index element={<Navigate to="dashboard/me" replace />} />
            <Route path="dashboard/:view" element={<DashboardPage />} />
            <Route path="employees" element={<EmployeesPage />} />
            <Route path="employees/:employeeId" element={<EmployeeDetailPage />} />
            <Route path="teams" element={<TeamsPage />} />
            <Route path="teams/:teamId" element={<TeamDetailPage />} />
            <Route path="projects" element={<ProjectsPage />} />
            <Route path="projects/:projectId" element={<ProjectDetailPage />} />
            <Route path="recruiting" element={<RecruitingPage />} />
            <Route path="recruiting/:jobOpeningId" element={<JobOpeningDetailPage />} />
            <Route path="one-on-ones/:entryId" element={<OneOnOneDetailPage />} />
            <Route path="discipline" element={<DisciplineListPage />} />
            <Route path="discipline/:caseId" element={<DisciplineDetailPage />} />
            <Route path="skills" element={<SkillsPage />} />
            <Route path="morale" element={<MoraleHistoryPage />} />
            <Route path="hardware" element={<HardwareListPage />} />
            <Route path="hardware/:hardwareId" element={<HardwareDetailPage />} />
            <Route path="softwares" element={<SoftwareListPage />} />
            <Route path="softwares/:softwareId" element={<SoftwareDetailPage />} />
            <Route path="adminland" element={<AdminlandPage />} />
          </Route>
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
