#import <AppKit/AppKit.h>
#import <stdlib.h>

#import "bridge.h"
#import "_cgo_export.h"

// Go returns owned C strings, including nil for successful operations without a value.
static NSString *TakeString(char *value) {
    if (!value) {
        return nil;
    }
    NSString *string = [NSString stringWithUTF8String:value];
    free(value);
    return string;
}

@interface EliseController : NSObject <NSApplicationDelegate, NSMenuDelegate>
@property(nonatomic, strong) NSStatusItem *statusItem;
@property(nonatomic, strong) NSMenu *menu;
@property(nonatomic, strong) NSArray<NSMenuItem *> *codeItems;
@property(nonatomic, strong) NSTimer *timer;
@property(nonatomic, copy) NSString *loadError;
@end

@implementation EliseController

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    self.statusItem = [NSStatusBar.systemStatusBar statusItemWithLength:NSVariableStatusItemLength];
    self.statusItem.button.title = @"ELISE OTP";
    self.statusItem.button.toolTip = @"Elise 验证码";
    self.menu = [[NSMenu alloc] initWithTitle:@"Elise"];
    self.menu.delegate = self;
    self.menu.autoenablesItems = NO;
    self.statusItem.menu = self.menu;

    NSMenu *mainMenu = [[NSMenu alloc] init];
    NSMenuItem *appItem = [[NSMenuItem alloc] initWithTitle:@"Elise" action:NULL keyEquivalent:@""];
    NSMenu *appMenu = [[NSMenu alloc] initWithTitle:@"Elise"];
    [appMenu addItemWithTitle:@"退出 Elise" action:@selector(terminate:) keyEquivalent:@"q"];
    appItem.submenu = appMenu;
    [mainMenu addItem:appItem];

    NSMenuItem *editItem = [[NSMenuItem alloc] initWithTitle:@"编辑" action:NULL keyEquivalent:@""];
    NSMenu *editMenu = [[NSMenu alloc] initWithTitle:@"编辑"];
    [editMenu addItemWithTitle:@"剪切" action:@selector(cut:) keyEquivalent:@"x"];
    [editMenu addItemWithTitle:@"复制" action:@selector(copy:) keyEquivalent:@"c"];
    [editMenu addItemWithTitle:@"粘贴" action:@selector(paste:) keyEquivalent:@"v"];
    [editMenu addItemWithTitle:@"全选" action:@selector(selectAll:) keyEquivalent:@"a"];
    editItem.submenu = editMenu;
    [mainMenu addItem:editItem];
    NSApp.mainMenu = mainMenu;

    self.loadError = TakeString(EliseLoad());
    [self rebuildMenu];
}

- (NSArray<NSDictionary *> *)snapshot {
    NSData *data = [TakeString(EliseSnapshot()) dataUsingEncoding:NSUTF8StringEncoding];
    return [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
}

- (NSMenuItem *)itemWithTitle:(NSString *)title action:(SEL)action {
    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:@""];
    item.target = self;
    item.enabled = (action != NULL);
    return item;
}

- (NSString *)codeTitle:(NSDictionary *)entry {
    return [NSString stringWithFormat:@"%@   %@", entry[@"name"], entry[@"code"]];
}

- (void)rebuildMenu {
    [self.menu removeAllItems];
    NSMutableArray<NSMenuItem *> *codeItems = [NSMutableArray array];

    if (self.loadError) {
        [self.menu addItem:[self itemWithTitle:@"无法读取本地账户" action:NULL]];
        [self.menu addItem:[self itemWithTitle:@"重新读取本地账户…" action:@selector(reload:)]];
    } else {
        NSArray<NSDictionary *> *entries = [self snapshot];
        NSMenu *removeMenu = [[NSMenu alloc] initWithTitle:@"移除账户"];
        removeMenu.autoenablesItems = NO;
        for (NSDictionary *entry in entries) {
            NSInteger index = (NSInteger)codeItems.count;
            NSMenuItem *code = [self itemWithTitle:[self codeTitle:entry] action:@selector(copyCode:)];
            code.toolTip = @"点击复制当前验证码";
            code.tag = index;
            [codeItems addObject:code];
            [self.menu addItem:code];

            NSMenuItem *remove = [self itemWithTitle:entry[@"name"] action:@selector(removeAccount:)];
            remove.tag = index;
            [removeMenu addItem:remove];
        }
        if (entries.count == 0) {
            [self.menu addItem:[self itemWithTitle:@"尚未添加账户" action:NULL]];
        }
        [self.menu addItem:NSMenuItem.separatorItem];
        [self.menu addItem:[self itemWithTitle:@"添加账户…" action:@selector(addAccount:)]];
        if (entries.count > 0) {
            NSMenuItem *remove = [self itemWithTitle:@"移除账户" action:NULL];
            remove.submenu = removeMenu;
            remove.enabled = YES;
            [self.menu addItem:remove];
        }
    }
    self.codeItems = codeItems;
    [self.menu addItem:NSMenuItem.separatorItem];
    [self.menu addItem:[self itemWithTitle:@"退出 Elise" action:@selector(quit:)]];
}

- (void)menuWillOpen:(NSMenu *)menu {
    [self refreshCodes:nil];
    if (self.codeItems.count > 0) {
        self.timer = [NSTimer timerWithTimeInterval:0.25 target:self
                                           selector:@selector(refreshCodes:) userInfo:nil repeats:YES];
        [NSRunLoop.mainRunLoop addTimer:self.timer forMode:NSRunLoopCommonModes];
        [NSRunLoop.mainRunLoop addTimer:self.timer forMode:NSEventTrackingRunLoopMode];
    }
}

- (void)menuDidClose:(NSMenu *)menu {
    [self.timer invalidate];
    self.timer = nil;
}

- (void)refreshCodes:(NSTimer *)timer {
    NSArray<NSDictionary *> *entries = [self snapshot];
    [self.codeItems enumerateObjectsUsingBlock:^(NSMenuItem *item, NSUInteger index, BOOL *stop) {
        if (index < entries.count) {
            NSString *title = [self codeTitle:entries[index]];
            if (![item.title isEqualToString:title]) {
                item.title = title;
            }
        }
    }];
}

- (void)activate {
    if (@available(macOS 14.0, *)) {
        [NSApp activate];
    } else {
        [NSApp activateIgnoringOtherApps:YES];
    }
}

- (void)showError:(NSString *)message {
    [self activate];
    NSAlert *alert = [[NSAlert alloc] init];
    alert.alertStyle = NSAlertStyleWarning;
    alert.messageText = @"Elise";
    alert.informativeText = message;
    [alert addButtonWithTitle:@"好"];
    [alert runModal];
}

- (void)reload:(id)sender {
    self.loadError = TakeString(EliseLoad());
    [self rebuildMenu];
    if (self.loadError) {
        [self showError:self.loadError];
    }
}

- (void)copyCode:(NSMenuItem *)sender {
    NSString *code = TakeString(EliseCopyCode((int)sender.tag));
    if (!code.length) {
        return;
    }
    NSPasteboard *clipboard = NSPasteboard.generalPasteboard;
    [clipboard clearContents];
    if (![clipboard setString:code forType:NSPasteboardTypeString]) {
        [self showError:@"无法将验证码复制到剪贴板。"];
    }
}

- (void)addAccount:(id)sender {
    [self activate];
    NSAlert *alert = [[NSAlert alloc] init];
    alert.messageText = @"添加 TOTP 账户";
    alert.informativeText = @"粘贴验证器链接（otpauth://totp）或填写 Base32 密钥。使用 Base32 密钥时需要填写账户名称。";
    [alert addButtonWithTitle:@"添加"];
    [alert addButtonWithTitle:@"取消"];

    NSTextField *name = [NSTextField textFieldWithString:@""];
    name.placeholderString = @"账户名称（可覆盖链接中的名称）";
    name.accessibilityLabel = @"账户名称";
    NSSecureTextField *secret = [[NSSecureTextField alloc] initWithFrame:NSZeroRect];
    secret.editable = YES;
    secret.bezeled = YES;
    secret.placeholderString = @"验证器链接或 Base32 密钥";
    secret.accessibilityLabel = @"验证器链接或 Base32 密钥";

    NSStackView *fields = [NSStackView stackViewWithViews:@[name, secret]];
    fields.orientation = NSUserInterfaceLayoutOrientationVertical;
    fields.alignment = NSLayoutAttributeLeading;
    fields.spacing = 8;
    [name.widthAnchor constraintEqualToConstant:320].active = YES;
    [secret.widthAnchor constraintEqualToAnchor:name.widthAnchor].active = YES;
    fields.frame = NSMakeRect(0, 0, 320, 56);
    alert.accessoryView = fields;
    [alert layout];
    name.nextKeyView = secret;
    [alert.window setInitialFirstResponder:name];

    while ([alert runModal] == NSAlertFirstButtonReturn) {
        NSString *error = TakeString(EliseAdd((char *)secret.stringValue.UTF8String,
                                             (char *)name.stringValue.UTF8String));
        if (!error) {
            secret.stringValue = @"";
            [self rebuildMenu];
            return;
        }
        [self showError:error];
    }
    secret.stringValue = @"";
}

- (void)removeAccount:(NSMenuItem *)sender {
    [self activate];
    NSAlert *alert = [[NSAlert alloc] init];
    alert.alertStyle = NSAlertStyleWarning;
    alert.messageText = @"移除此账户？";
    alert.informativeText = [NSString stringWithFormat:@"Elise 将删除本地保存的「%@」账户。", sender.title];
    [alert addButtonWithTitle:@"取消"];
    [alert addButtonWithTitle:@"移除"];
    if ([alert runModal] != NSAlertSecondButtonReturn) {
        return;
    }
    NSString *error = TakeString(EliseRemove((int)sender.tag));
    if (error) {
        [self showError:error];
        return;
    }
    [self rebuildMenu];
}

- (void)quit:(id)sender {
    [NSApp terminate:sender];
}

@end

void EliseRunApp(void) {
    @autoreleasepool {
        NSApplication *app = NSApplication.sharedApplication;
        [app setActivationPolicy:NSApplicationActivationPolicyAccessory];
        __attribute__((objc_precise_lifetime)) EliseController *controller = [[EliseController alloc] init];
        app.delegate = controller;
        [app run];
    }
}
