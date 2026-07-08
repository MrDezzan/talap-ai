import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Sidebar from '../components/Sidebar';
import { useMobile } from '../hooks/useMobile';

export default function AppLayout() {
  const { user, aiProfile, aiProfileLoaded } = useAuth();
  const isMobile = useMobile();
  const location = useLocation();
  
  if (!user) return <Navigate to="/login" replace />;

  // Wait until profile loaded check is completed
  if (!aiProfileLoaded) {
    return (
      <div style={{ display: 'flex', height: '100vh', alignItems: 'center', justifyContent: 'center', background: '#F5F7FB', fontFamily: 'var(--font-sans)', fontSize: 16, color: '#5A6485' }}>
        Загрузка профиля...
      </div>
    );
  }

  const hasTestResult = aiProfile && aiProfile.top_profession;
  const isQuizRoute = location.pathname === '/quiz';
  const isSettingsRoute = location.pathname === '/settings';

  // Force users to take the quiz if they haven't completed it yet
  if (!hasTestResult && !isQuizRoute && !isSettingsRoute) {
    return <Navigate to="/quiz" replace />;
  }

  // Hide navigation if the user hasn't taken the quiz and is on the quiz screen
  const showNavigation = hasTestResult;

  return (
    <div style={{ 
      display: 'flex', 
      height: '100vh', 
      flexDirection: isMobile ? 'column' : 'row',
      overflow: 'hidden',
      background: '#F5F7FB'
    }}>
      {!isMobile && showNavigation && <Sidebar />}
      <main style={{ 
        flex: 1, 
        display: 'flex', 
        flexDirection: 'column', 
        overflowY: 'auto',
        paddingBottom: isMobile && showNavigation ? 64 : 0,
        minWidth: 0,
      }}>
        <Outlet />
      </main>
      {isMobile && showNavigation && <Sidebar />}
    </div>
  );
}
