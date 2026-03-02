import { useEffect } from "preact/hooks";
import { useLocation } from "preact-iso";
import { useAuth } from "../hooks/use-auth";

export function LogoutPage(_props: { path?: string }) {
  const { logout } = useAuth();
  const { route } = useLocation();

  useEffect(() => {
    logout().then(() => route("/"));
  }, [logout, route]);

  return (
    <div class="flex justify-center py-16">
      <div class="text-white/30 text-sm">ログアウト中...</div>
    </div>
  );
}
