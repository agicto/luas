import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Switch } from '@/components/ui/switch';
import { ApiKeyPanel } from '@/features/api-key';
import { getT } from '@/i18n/server';

/**
 * Replaceable settings surface for the starter console.
 */
export default async function SettingsPage() {
  const t = await getT('settings');

  return (
    <div className="flex-1 space-y-4 p-6 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">{t('title')}</h2>
      </div>

      <Tabs defaultValue="general" className="space-y-4">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger value="general">{t('tabs.general')}</TabsTrigger>
          <TabsTrigger value="notifications">{t('tabs.notifications')}</TabsTrigger>
          <TabsTrigger value="security">{t('tabs.security')}</TabsTrigger>
          <TabsTrigger value="api">{t('tabs.api')}</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t('system.title')}</CardTitle>
              <CardDescription>{t('system.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="company-name">{t('system.companyName')}</Label>
                <Input
                  id="company-name"
                  placeholder={t('system.companyPlaceholder')}
                  defaultValue={t('system.companyDefault')}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="site-url">{t('system.websiteUrl')}</Label>
                <Input
                  id="site-url"
                  placeholder={t('system.websitePlaceholder')}
                  defaultValue="https://example.com"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="support-email">{t('system.supportEmail')}</Label>
                <Input
                  id="support-email"
                  placeholder={t('system.supportEmailPlaceholder')}
                  defaultValue="support@example.com"
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button>{t('system.save')}</Button>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('display.title')}</CardTitle>
              <CardDescription>{t('display.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="dark-mode" className="text-base">
                    {t('display.darkMode')}
                  </Label>
                  <p id="dark-mode-description" className="text-sm text-muted-foreground">
                    {t('display.darkModeDescription')}
                  </p>
                </div>
                <Switch id="dark-mode" aria-describedby="dark-mode-description" />
              </div>
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="automatic-theme" className="text-base">
                    {t('display.autoTheme')}
                  </Label>
                  <p id="automatic-theme-description" className="text-sm text-muted-foreground">
                    {t('display.autoThemeDescription')}
                  </p>
                </div>
                <Switch
                  id="automatic-theme"
                  aria-describedby="automatic-theme-description"
                  defaultChecked
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button>{t('display.save')}</Button>
            </CardFooter>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t('notifications.title')}</CardTitle>
              <CardDescription>{t('notifications.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="email-notifications" className="text-base">
                    {t('notifications.email')}
                  </Label>
                  <p id="email-notifications-description" className="text-sm text-muted-foreground">
                    {t('notifications.emailDescription')}
                  </p>
                </div>
                <Switch
                  id="email-notifications"
                  aria-describedby="email-notifications-description"
                  defaultChecked
                />
              </div>
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="sms-notifications" className="text-base">
                    {t('notifications.sms')}
                  </Label>
                  <p id="sms-notifications-description" className="text-sm text-muted-foreground">
                    {t('notifications.smsDescription')}
                  </p>
                </div>
                <Switch id="sms-notifications" aria-describedby="sms-notifications-description" />
              </div>
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="browser-push" className="text-base">
                    {t('notifications.browserPush')}
                  </Label>
                  <p id="browser-push-description" className="text-sm text-muted-foreground">
                    {t('notifications.browserPushDescription')}
                  </p>
                </div>
                <Switch
                  id="browser-push"
                  aria-describedby="browser-push-description"
                  defaultChecked
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button>{t('notifications.save')}</Button>
            </CardFooter>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t('security.title')}</CardTitle>
              <CardDescription>{t('security.description')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between gap-4">
                <div className="space-y-0.5">
                  <Label htmlFor="two-factor" className="text-base">
                    {t('security.twoFactor')}
                  </Label>
                  <p id="two-factor-description" className="text-sm text-muted-foreground">
                    {t('security.twoFactorDescription')}
                  </p>
                </div>
                <Switch id="two-factor" aria-describedby="two-factor-description" defaultChecked />
              </div>
              <div className="space-y-2">
                <Label htmlFor="session-timeout">{t('security.sessionTimeout')}</Label>
                <Input id="session-timeout" type="number" defaultValue="30" />
              </div>
              <Button variant="outline">{t('security.changePassword')}</Button>
            </CardContent>
            <CardFooter>
              <Button>{t('security.save')}</Button>
            </CardFooter>
          </Card>
        </TabsContent>

        <TabsContent value="api" className="space-y-4">
          <ApiKeyPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}
