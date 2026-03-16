import { Router, Route } from '@solidjs/router';
import Sidebar from './components/layout/Sidebar';
import EtlList from './pages/EtlList';
import EtlDetail from './pages/EtlDetail';
import EtlCreate from './pages/EtlCreate';
import Connections from './pages/Connections';

export default function App() {
  return (
    <Router
      root={(props) => (
        <div class="flex h-screen bg-white dark:bg-gray-900">
          <Sidebar />
          <main class="flex-1 flex flex-col overflow-hidden">
            {props.children}
          </main>
        </div>
      )}
    >
      <Route path="/" component={EtlList} />
      <Route path="/new" component={EtlCreate} />
      <Route path="/etl/:name" component={EtlDetail} />
      <Route path="/connections" component={Connections} />
    </Router>
  );
}
