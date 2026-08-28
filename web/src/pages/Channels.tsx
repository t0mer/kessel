import { useState } from "react";
import { Plus, Trash2, Send, Pencil } from "lucide-react";
import { api } from "@/lib/api";
import { useApi } from "@/lib/hooks";
import type { Channel, ChannelType } from "@/lib/types";
import { PageHeader } from "@/components/Layout";
import { Badge, Button, Card, EmptyState, ErrorState, Field, Input, Modal, Select, Spinner, Toggle } from "@/components/ui";

const TYPE_LABELS: Record<ChannelType, string> = {
  shoutrrr: "Shoutrrr",
  greenapi: "GreenAPI (WhatsApp)",
  whatsapp_web: "WhatsApp multi-device",
};

export function Channels() {
  const { data, error, loading, reload } = useApi<Channel[]>(() => api.listChannels(), [], 0);
  const [editing, setEditing] = useState<Channel | null>(null);
  const [adding, setAdding] = useState(false);

  return (
    <>
      <PageHeader
        title="Channels"
        subtitle="Where Kessel sends threshold alerts"
        action={<Button onClick={() => setAdding(true)}><Plus size={16} /> Add channel</Button>}
      />
      {loading && <Spinner />}
      {error && <ErrorState message={error} />}
      {data && data.length === 0 && (
        <EmptyState title="No channels yet" hint="Add a channel to receive alerts when scores cross your thresholds." action={<Button onClick={() => setAdding(true)}>Add channel</Button>} />
      )}
      {data && data.length > 0 && (
        <div className="space-y-3">
          {data.map((c) => (
            <Card key={c.id} className="flex flex-wrap items-center gap-4 p-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-fg">{c.name}</span>
                  <Badge tone="primary">{TYPE_LABELS[c.type] ?? c.type}</Badge>
                </div>
                <div className="mt-1 flex gap-2 text-xs text-muted">
                  {c.notify_on_success && <span>on success</span>}
                  {c.notify_on_failure && <span>on failure</span>}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <Toggle checked={c.enabled} label="Enabled" onChange={async (v) => { await api.updateChannel(c.id, { enabled: v, type: c.type }); reload(); }} />
                <Button size="sm" variant="ghost" aria-label="Edit" onClick={() => setEditing(c)}><Pencil size={15} /></Button>
                <Button size="sm" variant="ghost" aria-label="Delete" onClick={async () => { if (confirm(`Delete ${c.name}?`)) { await api.deleteChannel(c.id); reload(); } }}><Trash2 size={15} /></Button>
              </div>
            </Card>
          ))}
        </div>
      )}
      {(adding || editing) && (
        <ChannelForm channel={editing} onClose={() => { setAdding(false); setEditing(null); }} onSaved={() => { setAdding(false); setEditing(null); reload(); }} />
      )}
    </>
  );
}

function ChannelForm({ channel, onClose, onSaved }: { channel: Channel | null; onClose: () => void; onSaved: () => void }) {
  const [type, setType] = useState<ChannelType>(channel?.type ?? "shoutrrr");
  const [name, setName] = useState(channel?.name ?? "");
  const [onSuccess, setOnSuccess] = useState(channel?.notify_on_success ?? false);
  const [onFailure, setOnFailure] = useState(channel?.notify_on_failure ?? true);

  // provider fields
  const [url, setUrl] = useState("");
  const [instanceId, setInstanceId] = useState("");
  const [token, setToken] = useState("");
  const [phone, setPhone] = useState("");
  const [apiUrl, setApiUrl] = useState("");

  const [err, setErr] = useState<string>();
  const [testMsg, setTestMsg] = useState<string>();
  const [saving, setSaving] = useState(false);

  function buildConfig(): Record<string, string> {
    if (type === "greenapi") return { instance_id: instanceId.trim(), token: token.trim(), phone: phone.trim(), api_url: apiUrl.trim() };
    return { url: url.trim() };
  }

  async function sendTest() {
    setTestMsg(undefined);
    setErr(undefined);
    try {
      await api.testChannel({ type, config: buildConfig() });
      setTestMsg("Test message sent.");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Test failed.");
    }
  }

  async function save() {
    setSaving(true);
    setErr(undefined);
    try {
      const base = { type, name, notify_on_success: onSuccess, notify_on_failure: onFailure };
      if (channel) {
        // Only include config if the user re-entered it (edit keeps existing when omitted).
        const hasConfig = type === "greenapi" ? instanceId || token || phone : url;
        await api.updateChannel(channel.id, hasConfig ? { ...base, config: buildConfig() } : base);
      } else {
        await api.createChannel({ ...base, config: buildConfig() });
      }
      onSaved();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal title={channel ? "Edit channel" : "Add channel"} onClose={onClose}>
      <div className="space-y-4">
        <Field label="Provider">
          <Select value={type} onChange={(e) => setType(e.target.value as ChannelType)} disabled={!!channel}>
            <option value="shoutrrr">Shoutrrr (Slack, Discord, Telegram, email…)</option>
            <option value="greenapi">GreenAPI (WhatsApp cloud)</option>
            <option value="whatsapp_web" disabled>WhatsApp multi-device (coming soon)</option>
          </Select>
        </Field>
        <Field label="Name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Team Slack" />
        </Field>

        {type === "shoutrrr" && (
          <Field label="Shoutrrr URL" hint={channel ? "Leave blank to keep the saved URL." : "e.g. slack://token@channel or telegram://token@telegram?chats=@name"}>
            <Input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="slack://token@channel" />
          </Field>
        )}
        {type === "greenapi" && (
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Instance ID"><Input value={instanceId} onChange={(e) => setInstanceId(e.target.value)} /></Field>
            <Field label="Token"><Input value={token} onChange={(e) => setToken(e.target.value)} /></Field>
            <Field label="Recipient phone" hint="International, digits only (e.g. 972501234567)."><Input value={phone} onChange={(e) => setPhone(e.target.value)} /></Field>
            <Field label="API URL (optional)" hint="Leave blank for api.green-api.com."><Input value={apiUrl} onChange={(e) => setApiUrl(e.target.value)} placeholder="https://api.green-api.com" /></Field>
          </div>
        )}

        <div className="flex flex-wrap gap-6">
          <label className="flex items-center gap-2 text-sm text-fg"><Toggle checked={onSuccess} onChange={setOnSuccess} /> Notify on success</label>
          <label className="flex items-center gap-2 text-sm text-fg"><Toggle checked={onFailure} onChange={setOnFailure} /> Notify on failure</label>
        </div>

        {testMsg && <div className="rounded-lg border border-good/40 bg-good/10 px-3 py-2 text-sm text-good">{testMsg}</div>}
        {err && <ErrorState message={err} />}

        <div className="flex items-center justify-between pt-2">
          <Button variant="outline" onClick={sendTest}><Send size={15} /> Send test</Button>
          <div className="flex gap-2">
            <Button variant="ghost" onClick={onClose}>Cancel</Button>
            <Button onClick={save} disabled={saving}>{saving ? "Saving…" : "Save"}</Button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
